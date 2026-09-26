package service

import (
	"context"
	"fmt"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/model"

	"go.mongodb.org/mongo-driver/bson"
)

type EpochService struct {
	config   *config.Config
	collName string
}

func NewEpochService(config *config.Config) *EpochService {
	return &EpochService{config: config, collName: "epochs"}
}

func (s *EpochService) Sync(ctx context.Context) error {
	client, err := getMongoClient(ctx)
	if err != nil {
		return err
	}
	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	head, err := s.currentEpoch(ctx)
	if err != nil {
		return err
	}

	cursor := head
	upserted := 0
	for {
		url := fmt.Sprintf("%s/api/v1/epochs?epoch=%d&limit=100", s.config.Dora.URL, cursor)
		payload, err := fetchJSON(ctx, url)
		if err != nil {
			return err
		}
		data, err := dataObject(payload)
		if err != nil {
			return err
		}
		raw, ok := data["epochs"].([]any)
		if !ok {
			return fmt.Errorf("epochs payload missing epochs array")
		}
		if len(raw) == 0 {
			break
		}

		minEpoch := cursor
		for _, item := range raw {
			e, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("epoch item is not an object")
			}
			epochNum := toInt(e["epoch"])
			if epochNum < minEpoch {
				minEpoch = epochNum
			}
			doc := model.Epoch{
				Model:             s.config.Model,
				Epoch:             epochNum,
				Finalized:         toBool(e["finalized"]),
				VotingFinalized:   toBool(e["voting_finalized"]),
				VotingJustified:   toBool(e["voting_justified"]),
				Validators:        toInt(e["validators"]),
				ValidatorBalance:  toString(e["validator_balance"]),
				EligibleEther:     toString(e["eligible_ether"]),
				TargetVoted:       toInt(e["target_voted"]),
				HeadVoted:         toInt(e["head_voted"]),
				TotalVoted:        toInt(e["total_voted"]),
				VoteParticipation: toInt(e["vote_participation"]),
				Attestations:      toInt(e["attestations"]),
				Deposits:          toInt(e["deposits"]),
				DepositsAmount:    toString(e["deposits_amount"]),
				ProposerSlashings: toInt(e["proposer_slashings"]),
				AttesterSlashings: toInt(e["attester_slashings"]),
				Exits:             toInt(e["exits"]),
				WithdrawalsCount:  toInt(e["withdrawals_count"]),
				WithdrawalsAmount: toString(e["withdrawals_amount"]),
				BlsChanges:        toInt(e["bls_changes"]),
				SyncParticipation: toInt(e["sync_participation"]),
				ProposedBlocks:    toInt(e["proposed_blocks"]),
				MissedBlocks:      toInt(e["missed_blocks"]),
				OrphanedBlocks:    toInt(e["orphaned_blocks"]),
			}
			filter := bson.M{"model": s.config.Model, "epoch": epochNum}
			inserted, err := upsertDocument(ctx, coll, filter, doc)
			if err != nil {
				return err
			}
			if inserted {
				upserted++
			}
		}

		if minEpoch <= 0 {
			break
		}
		next := minEpoch - 1
		if next >= cursor {
			return fmt.Errorf("epochs pagination did not move backward from %d", cursor)
		}
		cursor = next
	}

	if upserted > 0 {
		log.Printf("epochs: inserted %d (head %d)", upserted, head)
	}
	return nil
}

func (s *EpochService) currentEpoch(ctx context.Context) (int, error) {
	payload, err := fetchJSON(ctx, s.config.Dora.URL+"/api/v1/epochs?limit=1")
	if err != nil {
		return 0, err
	}
	return currentEpochFromPayload(payload)
}
