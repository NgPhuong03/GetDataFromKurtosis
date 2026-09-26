package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"mongo_fetch/config"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Proposal struct {
	Epoch     int
	Slot      int
	Validator int
}

type ValidatorStake struct {
	Index            int
	Epoch            int
	EffectiveBalance string
	Balance          string
}

type AggregateRow struct {
	Model          string
	ValidatorIndex int
	Stake          string
	ProposerCount  int
}

type RunStatus struct {
	Database         string `json:"database"`
	Model            string `json:"model"`
	TargetEpochs     int    `json:"target_epochs"`
	SlotsPerEpoch    int    `json:"slots_per_epoch"`
	CurrentEpoch     int    `json:"current_epoch"`
	ComparableEpochs int    `json:"comparable_epochs"`
	StoredSlots      int    `json:"stored_slots"`
	ValidatorCount   int    `json:"validator_count"`
	StakeUnit        string `json:"stake_unit"`
	StakeField       string `json:"stake_field"`
	MismatchedEpochs []int  `json:"mismatched_epochs,omitempty"`
	Complete         bool   `json:"complete"`
	RawCSV           string `json:"raw_csv"`
	AggregatedCSV    string `json:"aggregated_csv"`
}

// ComparableEpochs is the longest prefix of epochs, starting at 0, that each
// contain exactly slotsPerEpoch stored slots. A gap or a partial epoch ends the prefix.
func ComparableEpochs(slotCount map[int]int, slotsPerEpoch, target int) (int, []int) {
	if slotsPerEpoch <= 0 || target <= 0 {
		return 0, nil
	}
	var mismatches []int
	comparable := 0
	for epoch := 0; epoch < target; epoch++ {
		n := slotCount[epoch]
		if n != slotsPerEpoch {
			if n != 0 {
				mismatches = append(mismatches, epoch)
			}
			return comparable, mismatches
		}
		comparable = epoch + 1
	}
	return comparable, mismatches
}

// Aggregate counts proposers in epochs [0, maxEpochExclusive) and attaches the
// earliest effective balance Dora reported for each validator. Stake is gwei.
func Aggregate(modelName string, slots []Proposal, validators []ValidatorStake, maxEpochExclusive int) []AggregateRow {
	counts := map[int]int{}
	for _, slot := range slots {
		if slot.Epoch < 0 || slot.Epoch >= maxEpochExclusive {
			continue
		}
		counts[slot.Validator]++
	}

	type snap struct {
		epoch int
		stake string
	}
	stakes := map[int]snap{}
	for _, v := range validators {
		stake := v.EffectiveBalance
		if stake == "" {
			stake = v.Balance
		}
		prev, ok := stakes[v.Index]
		if ok && prev.epoch <= v.Epoch {
			continue
		}
		stakes[v.Index] = snap{epoch: v.Epoch, stake: stake}
	}

	indexes := map[int]struct{}{}
	for index := range counts {
		indexes[index] = struct{}{}
	}
	for index := range stakes {
		indexes[index] = struct{}{}
	}
	ordered := make([]int, 0, len(indexes))
	for index := range indexes {
		ordered = append(ordered, index)
	}
	sort.Ints(ordered)

	rows := make([]AggregateRow, 0, len(ordered))
	for _, index := range ordered {
		rows = append(rows, AggregateRow{
			Model:          modelName,
			ValidatorIndex: index,
			Stake:          stakes[index].stake,
			ProposerCount:  counts[index],
		})
	}
	return rows
}

// ExportDatabase writes CSV files for every weighting model stored in cfg.Mongo.DBName.
func ExportDatabase(ctx context.Context, cfg *config.Config) error {
	client, err := getMongoClient(ctx)
	if err != nil {
		return err
	}
	db := client.Database(cfg.Mongo.DBName)
	models, err := listModels(ctx, db)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return fmt.Errorf("database %q has no documents with a model", cfg.Mongo.DBName)
	}

	log.Printf("export database=%s models=%v", cfg.Mongo.DBName, models)
	for _, modelName := range models {
		modelCfg := *cfg
		modelCfg.Model = modelName
		status, err := Export(ctx, &modelCfg)
		if err != nil {
			return fmt.Errorf("export %s from %s: %w", modelName, cfg.Mongo.DBName, err)
		}
		log.Printf("database=%s %s raw=%s aggregated=%s", cfg.Mongo.DBName, statusLine(status), status.RawCSV, status.AggregatedCSV)
	}
	return nil
}

func listModels(ctx context.Context, db *mongo.Database) ([]string, error) {
	seen := map[string]struct{}{}
	for _, name := range []string{"slots", "validators", "epochs"} {
		values, err := db.Collection(name).Distinct(ctx, "model", bson.M{})
		if err != nil {
			return nil, err
		}
		for _, value := range values {
			modelName, ok := value.(string)
			if ok && modelName != "" {
				seen[modelName] = struct{}{}
			}
		}
	}
	models := make([]string, 0, len(seen))
	for modelName := range seen {
		models = append(models, modelName)
	}
	sort.Strings(models)
	return models, nil
}

func Progress(ctx context.Context, cfg *config.Config) (RunStatus, error) {
	slots, validators, currentEpoch, err := loadModel(ctx, cfg)
	if err != nil {
		return RunStatus{}, err
	}
	return statusFor(cfg, slots, validators, currentEpoch), nil
}

func Export(ctx context.Context, cfg *config.Config) (RunStatus, error) {
	slots, validators, currentEpoch, err := loadModel(ctx, cfg)
	if err != nil {
		return RunStatus{}, err
	}
	comparable, _ := countEpochs(slots, cfg)
	aggregated := Aggregate(cfg.Model, slots, validators, comparable)

	dir := filepath.Join(cfg.OutputDir, cfg.Mongo.DBName, cfg.Model)
	rawPath := filepath.Join(dir, "proposers_raw.csv")
	aggPath := filepath.Join(dir, "proposers_aggregated.csv")
	statusPath := filepath.Join(dir, "run_status.json")

	if err := writeRawCSV(rawPath, cfg.Model, slots); err != nil {
		return RunStatus{}, err
	}
	if err := writeAggregateCSV(aggPath, aggregated); err != nil {
		return RunStatus{}, err
	}

	status := statusFor(cfg, slots, validators, currentEpoch)
	status.RawCSV = rawPath
	status.AggregatedCSV = aggPath
	if err := writeJSON(statusPath, status); err != nil {
		return RunStatus{}, err
	}
	return status, nil
}

func loadModel(ctx context.Context, cfg *config.Config) ([]Proposal, []ValidatorStake, int, error) {
	client, err := getMongoClient(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	db := client.Database(cfg.Mongo.DBName)
	slots, err := loadProposals(ctx, db.Collection("slots"), cfg.Model)
	if err != nil {
		return nil, nil, 0, err
	}
	validators, err := loadValidatorStakes(ctx, db.Collection("validators"), cfg.Model)
	if err != nil {
		return nil, nil, 0, err
	}
	currentEpoch, err := loadHeadEpoch(ctx, db.Collection("epochs"), cfg.Model)
	if err != nil {
		return nil, nil, 0, err
	}
	return slots, validators, currentEpoch, nil
}

func countEpochs(slots []Proposal, cfg *config.Config) (int, []int) {
	slotCount := map[int]int{}
	for _, slot := range slots {
		slotCount[slot.Epoch]++
	}
	return ComparableEpochs(slotCount, cfg.SlotsPerEpoch, cfg.TargetEpochs)
}

func statusFor(cfg *config.Config, slots []Proposal, validators []ValidatorStake, currentEpoch int) RunStatus {
	comparable, mismatches := countEpochs(slots, cfg)
	return RunStatus{
		Database:         cfg.Mongo.DBName,
		Model:            cfg.Model,
		TargetEpochs:     cfg.TargetEpochs,
		SlotsPerEpoch:    cfg.SlotsPerEpoch,
		CurrentEpoch:     currentEpoch,
		ComparableEpochs: comparable,
		StoredSlots:      len(slots),
		ValidatorCount:   distinctValidators(validators),
		StakeUnit:        "gwei",
		StakeField:       "effective_balance",
		MismatchedEpochs: mismatches,
		Complete:         comparable >= cfg.TargetEpochs,
	}
}

func distinctValidators(validators []ValidatorStake) int {
	seen := map[int]struct{}{}
	for _, v := range validators {
		seen[v.Index] = struct{}{}
	}
	return len(seen)
}

type slotKey struct {
	Epoch    int `bson:"epoch"`
	Slot     int `bson:"slot"`
	Proposer int `bson:"proposer"`
}

type validatorKey struct {
	Index            int    `bson:"index"`
	Epoch            int    `bson:"epoch"`
	EffectiveBalance string `bson:"effective_balance"`
	Balance          string `bson:"balance"`
}

func loadProposals(ctx context.Context, coll *mongo.Collection, modelName string) ([]Proposal, error) {
	opts := options.Find().SetProjection(bson.M{"epoch": 1, "slot": 1, "proposer": 1, "_id": 0})
	cursor, err := coll.Find(ctx, bson.M{"model": modelName}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var slots []Proposal
	for cursor.Next(ctx) {
		var row slotKey
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		slots = append(slots, Proposal{Epoch: row.Epoch, Slot: row.Slot, Validator: row.Proposer})
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	sort.Slice(slots, func(i, j int) bool {
		if slots[i].Epoch == slots[j].Epoch {
			return slots[i].Slot < slots[j].Slot
		}
		return slots[i].Epoch < slots[j].Epoch
	})
	return slots, nil
}

func loadValidatorStakes(ctx context.Context, coll *mongo.Collection, modelName string) ([]ValidatorStake, error) {
	opts := options.Find().SetProjection(bson.M{
		"index": 1, "epoch": 1, "effective_balance": 1, "balance": 1, "_id": 0,
	})
	cursor, err := coll.Find(ctx, bson.M{"model": modelName}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var validators []ValidatorStake
	for cursor.Next(ctx) {
		var row validatorKey
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		validators = append(validators, ValidatorStake{
			Index:            row.Index,
			Epoch:            row.Epoch,
			EffectiveBalance: row.EffectiveBalance,
			Balance:          row.Balance,
		})
	}
	return validators, cursor.Err()
}

func loadHeadEpoch(ctx context.Context, coll *mongo.Collection, modelName string) (int, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "epoch", Value: -1}}).SetProjection(bson.M{"epoch": 1, "_id": 0})
	var row struct {
		Epoch int `bson:"epoch"`
	}
	err := coll.FindOne(ctx, bson.M{"model": modelName}, opts).Decode(&row)
	if err == mongo.ErrNoDocuments {
		return -1, nil
	}
	if err != nil {
		return 0, err
	}
	return row.Epoch, nil
}

func writeRawCSV(path, modelName string, slots []Proposal) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"model", "epoch", "slot", "validator_index"}); err != nil {
		return err
	}
	for _, slot := range slots {
		if err := w.Write([]string{
			modelName,
			strconv.Itoa(slot.Epoch),
			strconv.Itoa(slot.Slot),
			strconv.Itoa(slot.Validator),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func writeAggregateCSV(path string, rows []AggregateRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write([]string{"model", "validator_index", "stake", "proposer_count"}); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write([]string{
			row.Model,
			strconv.Itoa(row.ValidatorIndex),
			row.Stake,
			strconv.Itoa(row.ProposerCount),
		}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func statusLine(status RunStatus) string {
	return fmt.Sprintf(
		"model=%s comparable_epochs=%d/%d stored_slots=%d validators=%d complete=%v",
		status.Model,
		status.ComparableEpochs,
		status.TargetEpochs,
		status.StoredSlots,
		status.ValidatorCount,
		status.Complete,
	)
}
