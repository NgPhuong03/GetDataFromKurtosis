package service

import (
	"context"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type EpochService struct {
	config   *config.Config
	collName string
}

func NewEpochService(config *config.Config) *EpochService {
	return &EpochService{config: config, collName: "epochs"}
}

func (s *EpochService) Start() {
	log.Println("Starting epochs service")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := connectMongo(ctx)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}
	log.Println("Connected to mongo")
	defer func() {
		_ = client.Disconnect(context.Background())
	}()

	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	url := s.config.Dora.URL + "/api/v1/epochs"
	payload, err := fetchJSON(ctx, url)
	if err != nil {
		log.Fatalf("failed to fetch JSON: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		log.Fatalf("invalid payload['data'] shape")
	}

	rawEpochs, ok := data["epochs"].([]any)
	if !ok {
		log.Fatalf("failed to parse 'epochs' array")
	}

	arrCount := []int{0, 0, 0}
	arrCount[0] = len(rawEpochs)

	for _, item := range rawEpochs {
		e, ok := item.(map[string]any)
		if !ok {
			log.Fatalf("epoch item is not an object")
		}

		epochNum := toInt(e["epoch"])
		filter := bson.M{"epoch": epochNum}

		var existing model.Epoch
		err := coll.FindOne(ctx, filter).Decode(&existing)
		if err == mongo.ErrNoDocuments {
			epoch := model.Epoch{
				Epoch:              epochNum,
				Finalized:          toBool(e["finalized"]),
				VotingFinalized:    toBool(e["voting_finalized"]),
				VotingJustified:    toBool(e["voting_justified"]),
				Validators:         toInt(e["validators"]),
				ValidatorBalance:   toString(e["validator_balance"]),
				EligibleEther:      toString(e["eligible_ether"]),
				TargetVoted:        toInt(e["target_voted"]),
				HeadVoted:          toInt(e["head_voted"]),
				TotalVoted:         toInt(e["total_voted"]),
				VoteParticipation:  toInt(e["vote_participation"]),
				Attestations:       toInt(e["attestations"]),
				Deposits:           toInt(e["deposits"]),
				DepositsAmount:     toString(e["deposits_amount"]),
				ProposerSlashings:  toInt(e["proposer_slashings"]),
				AttesterSlashings:  toInt(e["attester_slashings"]),
				Exits:              toInt(e["exits"]),
				WithdrawalsCount:   toInt(e["withdrawals_count"]),
				WithdrawalsAmount:  toString(e["withdrawals_amount"]),
				BlsChanges:         toInt(e["bls_changes"]),
				SyncParticipation:  toInt(e["sync_participation"]),
				ProposedBlocks:     toInt(e["proposed_blocks"]),
				MissedBlocks:       toInt(e["missed_blocks"]),
				OrphanedBlocks:     toInt(e["orphaned_blocks"]),
			}
			insertedID, err := saveDocument(ctx, coll, epoch)
			if err != nil {
				log.Fatalf("failed to save document: %v", err)
			}
			log.Printf("saved document with _id=%v to %s.%s", insertedID, s.config.Mongo.DBName, s.collName)
			arrCount[2]++
		} else if err != nil {
			log.Fatalf("failed to find epoch: %v", err)
		} else {
			update := bson.M{
				"$set": bson.M{
					"epoch":               epochNum,
					"finalized":           toBool(e["finalized"]),
					"voting_finalized":    toBool(e["voting_finalized"]),
					"voting_justified":    toBool(e["voting_justified"]),
					"validators":          toInt(e["validators"]),
					"validator_balance":   toString(e["validator_balance"]),
					"eligible_ether":      toString(e["eligible_ether"]),
					"target_voted":        toInt(e["target_voted"]),
					"head_voted":          toInt(e["head_voted"]),
					"total_voted":         toInt(e["total_voted"]),
					"vote_participation":  toInt(e["vote_participation"]),
					"attestations":        toInt(e["attestations"]),
					"deposits":            toInt(e["deposits"]),
					"deposits_amount":     toString(e["deposits_amount"]),
					"proposer_slashings":  toInt(e["proposer_slashings"]),
					"attester_slashings":  toInt(e["attester_slashings"]),
					"exits":               toInt(e["exits"]),
					"withdrawals_count":   toInt(e["withdrawals_count"]),
					"withdrawals_amount":  toString(e["withdrawals_amount"]),
					"bls_changes":         toInt(e["bls_changes"]),
					"sync_participation":  toInt(e["sync_participation"]),
					"proposed_blocks":     toInt(e["proposed_blocks"]),
					"missed_blocks":       toInt(e["missed_blocks"]),
					"orphaned_blocks":     toInt(e["orphaned_blocks"]),
				},
			}
			_, err := coll.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Fatalf("failed to update document: %v", err)
			}
			log.Printf("updated document with _id=%v to %s.%s", existing.Epoch, s.config.Mongo.DBName, s.collName)
			arrCount[1]++
		}
	}

	log.Printf("\t\t\t FINISHED FETCHING EPOCHS\n\n")
	log.Printf("\t\tUpdated: %v\t, Inserted: %v\t, Total: %v", arrCount[1], arrCount[2], arrCount[0])
}