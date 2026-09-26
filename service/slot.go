package service

import (
	"context"
	"fmt"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SlotService struct {
	config   *config.Config
	collName string
}

func NewSlotService(config *config.Config) *SlotService {
	return &SlotService{config: config, collName: "slots"}
}

// Sync stores one proposer row per elapsed slot for this weighting model.
//
// Dora lists slots newest-first. It asks the indexer for limit+1 rows, returns
// limit rows, and the next page starts limit+1 rows later, so one slot is on
// neither page. Which slot that is depends on limit. Sync fetches each missing
// slot from a page size that actually contains it, and keeps with_orphaned=0
// so the proposer is the canonical assignment.
func (s *SlotService) Sync(ctx context.Context) error {
	client, err := getMongoClient(ctx)
	if err != nil {
		return err
	}
	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	head, scheduled, err := s.chainHead(ctx)
	if err != nil {
		return err
	}
	elapsed := head
	if scheduled {
		elapsed = head - 1
	}
	if elapsed < 0 {
		return nil
	}

	stored, err := loadStoredSlots(ctx, coll, s.config.Model)
	if err != nil {
		return err
	}
	var missing []int
	for slot := 0; slot <= elapsed; slot++ {
		if _, ok := stored[slot]; !ok {
			missing = append(missing, slot)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	type pageKey struct {
		limit int
		page  int
	}
	seen := map[pageKey]struct{}{}
	var pages []pageKey
	for _, slot := range missing {
		distance := head - slot
		if distance < 0 {
			continue
		}
		limit := limitForDistance(distance)
		key := pageKey{limit: limit, page: distance / (limit + 1)}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		pages = append(pages, key)
		if len(pages) == 40 {
			break
		}
	}

	insertedTotal := 0
	for _, page := range pages {
		if err := ctx.Err(); err != nil {
			return err
		}
		inserted, err := s.savePage(ctx, coll, head, page.limit, page.page)
		if err != nil {
			return err
		}
		insertedTotal += inserted
	}
	if insertedTotal > 0 || len(missing) > 0 {
		log.Printf("slots: inserted %d, gap before this pass %d", insertedTotal, len(missing))
	}
	return nil
}

// limitForDistance picks a Dora page size whose body contains this distance
// from the chain head. Distance 0 is the newest slot.
func limitForDistance(distance int) int {
	for _, limit := range []int{100, 99, 98} {
		if distance%(limit+1) != limit {
			return limit
		}
	}
	return 100
}

func (s *SlotService) chainHead(ctx context.Context) (int, bool, error) {
	url := fmt.Sprintf("%s/api/v1/slots?limit=1&with_missing=1&with_orphaned=0", s.config.Dora.URL)
	payload, err := fetchJSON(ctx, url)
	if err != nil {
		return 0, false, err
	}
	data, err := dataObject(payload)
	if err != nil {
		return 0, false, err
	}
	raw, ok := data["slots"].([]any)
	if !ok {
		return 0, false, fmt.Errorf("slots payload missing slots array")
	}
	if len(raw) == 0 {
		return -1, false, nil
	}
	item, ok := raw[0].(map[string]any)
	if !ok {
		return 0, false, fmt.Errorf("slot item is not an object")
	}
	return toInt(item["slot"]), toBool(item["scheduled"]), nil
}

func (s *SlotService) savePage(ctx context.Context, coll *mongo.Collection, head, limit, page int) (int, error) {
	url := fmt.Sprintf("%s/api/v1/slots?limit=%d&with_missing=1&with_orphaned=0", s.config.Dora.URL, limit)
	if page > 0 {
		// Indexer page index is currentSlot-start_slot. Page p then starts p pages back.
		url += fmt.Sprintf("&start_slot=%d", head-page)
	}
	payload, err := fetchJSON(ctx, url)
	if err != nil {
		return 0, err
	}
	data, err := dataObject(payload)
	if err != nil {
		return 0, err
	}
	raw, ok := data["slots"].([]any)
	if !ok {
		return 0, fmt.Errorf("slots payload missing slots array")
	}

	insertedTotal := 0
	for _, item := range raw {
		e, ok := item.(map[string]any)
		if !ok {
			return insertedTotal, fmt.Errorf("slot item is not an object")
		}
		if toBool(e["scheduled"]) {
			continue
		}
		slotNum := toInt(e["slot"])
		doc := model.Slot{
			Model:                      s.config.Model,
			Slot:                       slotNum,
			Epoch:                      toInt(e["epoch"]),
			Time:                       toString(e["time"]),
			Finalized:                  toBool(e["finalized"]),
			Scheduled:                  false,
			Status:                     toString(e["status"]),
			Proposer:                   toInt(e["proposer"]),
			ProposerName:               toString(e["proposer_name"]),
			AttestationCount:           toInt(e["attestation_count"]),
			DepositCount:               toInt(e["deposit_count"]),
			ExitCount:                  toInt(e["exit_count"]),
			ProposerSlashingCount:      toInt(e["proposer_slashing_count"]),
			AttesterSlashingCount:      toInt(e["attester_slashing_count"]),
			SyncAggregateParticipation: toInt(e["sync_aggregate_participation"]),
			EthTransactionCount:        toInt(e["eth_transaction_count"]),
			BlobCount:                  toInt(e["blob_count"]),
			WithEthBlock:               toBool(e["with_eth_block"]),
			EthBlockNumber:             toInt(e["eth_block_number"]),
			Graffiti:                   toString(e["graffiti"]),
			GraffitiText:               toString(e["graffiti_text"]),
			ElExtraData:                toString(e["el_extra_data"]),
			GasUsed:                    toInt(e["gas_used"]),
			GasLimit:                   toInt(e["gas_limit"]),
			BlockSize:                  toInt(e["block_size"]),
			BlockRoot:                  toString(e["block_root"]),
			ParentRoot:                 toString(e["parent_root"]),
			StateRoot:                  toString(e["state_root"]),
			RecvDelay:                  toInt(e["recv_delay"]),
			IsMevBlock:                 toBool(e["is_mev_block"]),
		}
		inserted, err := upsertDocument(ctx, coll, bson.M{"model": s.config.Model, "slot": slotNum}, doc)
		if err != nil {
			return insertedTotal, err
		}
		if inserted {
			insertedTotal++
		}
	}
	return insertedTotal, nil
}

func loadStoredSlots(ctx context.Context, coll *mongo.Collection, modelName string) (map[int]struct{}, error) {
	opts := options.Find().SetProjection(bson.M{"slot": 1, "_id": 0})
	cursor, err := coll.Find(ctx, bson.M{"model": modelName}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	stored := map[int]struct{}{}
	for cursor.Next(ctx) {
		var row struct {
			Slot int `bson:"slot"`
		}
		if err := cursor.Decode(&row); err != nil {
			return nil, err
		}
		stored[row.Slot] = struct{}{}
	}
	return stored, cursor.Err()
}
