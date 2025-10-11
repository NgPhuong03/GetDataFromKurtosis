package service

import (
	"context"
	"log"
	"time"

	"mongo_fetch/config"
	"mongo_fetch/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)


type ValidatorsService struct {
	config *config.Config
	collName string
}

func NewValidatorsService(config *config.Config) *ValidatorsService {
	return &ValidatorsService{config: config, collName: "validators"}
}

func (s *ValidatorsService) Start() {
	log.Println("Starting validators service")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

  client, err := getMongoClient(ctx)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}
	log.Println("Connected to mongo")

	url := s.config.Dora.URL + "/api/v1/validators?limit=1000"
	payload, err := fetchJSON(ctx, url)
	if err != nil {
		log.Fatalf("failed to fetch JSON: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		log.Fatalf("invalid payload['data'] shape")
	}
	rawValidators, ok := data["validators"].([]any)
	if !ok {
		log.Fatalf("failed to parse 'validators' array")
	}

	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	arrCount := []int{0,0,0}
	arrCount[0] = len(rawValidators)

	currEpoch := getCurrentEpoch(ctx, s.config.Dora.URL)

	for _, item := range rawValidators {
		v, ok := item.(map[string]any)
		if !ok {
			log.Fatalf("validator item is not an object")
		}

		publicKey := toString(v["public_key"])
		filter := bson.M{"public_key": publicKey, "epoch": currEpoch}

		var existing model.Validator
		errFind := coll.FindOne(ctx, filter).Decode(&existing)
		if errFind == nil {
			update := bson.M{
				"$set": bson.M{
					"index":                  toInt(v["index"]),
					"name":                   toString(v["name"]),
					"balance":                toString(v["balance"]),
					"effective_balance":      toString(v["effective_balance"]),
					"status":                 toString(v["status"]),
					"activation_time":        toString(v["activation_time"]),
					"withdrawal_address":     toString(v["withdrawal_address"]),
					"withdrawal_credentials": toString(v["withdrawal_credentials"]),
					"validator_liveness":     toInt(v["validator_liveness"]),
					"validator_liveness_max": toInt(v["validator_liveness_max"]),
				},
			}
			_, err := coll.UpdateOne(ctx, filter, update)
			if err != nil {
				log.Fatalf("failed to update validator: %v", err)
			}
			// log.Printf("updated validator: %v", v["index"])
			arrCount[1]++
		} else if errFind != mongo.ErrNoDocuments {
			log.Printf("failed to find validator: %v", errFind)
		} else {
			validator := model.Validator{
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
				Epoch:                 currEpoch,
			}

			_, err := saveDocument(ctx, coll, validator)
			if err != nil {
				log.Fatalf("failed to save document: %v", err)
			}
			// log.Printf("saved document with _id=%v to %s.%s", insertedID, s.config.Mongo.DBName, s.collName)
			arrCount[2]++
		}
	}
	log.Printf("\t\t\t FINISHED FETCHING VALIDATORS\n\n")
	log.Printf("\t\tUpdated: %v\t, Inserted: %v\t, Total: %v", arrCount[1], arrCount[2], arrCount[0])
}


func getCurrentEpoch(ctx context.Context, baseUrl string) int {
	url := baseUrl + "/api/v1/epochs?limit=1"
 	rest, err := fetchJSON(ctx, url)
	if err != nil {
		return -1
	}

	return toInt(rest["epoch"])
}