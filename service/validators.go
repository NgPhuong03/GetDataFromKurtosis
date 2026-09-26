package service

import (
	"context"
	"fmt"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/model"

	"go.mongodb.org/mongo-driver/bson"
)

type ValidatorsService struct {
	config   *config.Config
	collName string
}

func NewValidatorsService(config *config.Config) *ValidatorsService {
	return &ValidatorsService{config: config, collName: "validators"}
}

func (s *ValidatorsService) Sync(ctx context.Context) error {
	client, err := getMongoClient(ctx)
	if err != nil {
		return err
	}
	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	upserted := 0
	for page := 1; page < 10000; page++ {
		url := fmt.Sprintf("%s/api/v1/validators?limit=1000&page=%d", s.config.Dora.URL, page)
		payload, err := fetchJSON(ctx, url)
		if err != nil {
			return err
		}
		data, err := dataObject(payload)
		if err != nil {
			return err
		}
		epoch, err := requiredInt(data, "current_epoch")
		if err != nil {
			return fmt.Errorf("validators: %w", err)
		}
		raw, ok := data["validators"].([]any)
		if !ok {
			return fmt.Errorf("validators payload missing validators array")
		}

		for _, item := range raw {
			v, ok := item.(map[string]any)
			if !ok {
				return fmt.Errorf("validator item is not an object")
			}
			publicKey := toString(v["public_key"])
			doc := model.Validator{
				Model:                 s.config.Model,
				Index:                 toInt(v["index"]),
				Name:                  toString(v["name"]),
				PublicKey:             publicKey,
				Balance:               toString(v["balance"]),
				EffectiveBalance:      toString(v["effective_balance"]),
				Status:                toString(v["status"]),
				ActivationTime:        toString(v["activation_time"]),
				WithdrawalAddress:     toString(v["withdrawal_address"]),
				WithdrawalCredentials: toString(v["withdrawal_credentials"]),
				ValidatorLiveness:     toInt(v["validator_liveness"]),
				ValidatorLivenessMax:  toInt(v["validator_liveness_max"]),
				Epoch:                 epoch,
			}
			filter := bson.M{"model": s.config.Model, "public_key": publicKey, "epoch": epoch}
			inserted, err := upsertDocument(ctx, coll, filter, doc)
			if err != nil {
				return err
			}
			if inserted {
				upserted++
			}
		}

		totalPages, err := requiredInt(data, "total_pages")
		if err != nil || page >= totalPages || len(raw) == 0 {
			break
		}
	}

	if upserted > 0 {
		log.Printf("validators: inserted %d", upserted)
	}
	return nil
}
