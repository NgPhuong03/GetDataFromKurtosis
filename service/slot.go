package service

import (
	"context"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/model"
	"time"

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

func (s *SlotService) Start() {
	log.Println("Starting slots service")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

  client, err := getMongoClient(ctx)
	if err != nil {
		log.Fatalf("failed to connect mongo: %v", err)
	}
	log.Println("Connected to mongo")

	coll := client.Database(s.config.Mongo.DBName).Collection(s.collName)

	url := s.config.Dora.URL + "/api/v1/slots"
	payload, err := fetchJSON(ctx, url)
	if err != nil {
		log.Fatalf("failed to fetch JSON: %v", err)
	}

	data, ok := payload["data"].(map[string]any)
	if !ok {
		log.Fatalf("invalid payload['data'] shape")
	}

	arr := data["slots"].([]any)

	arrCount := []int{0, 0, 0}

	currentSlot, err := getCurrentSlot(arr)
	log.Println("Current slot", currentSlot)
	if err != nil {
		log.Fatalf("failed to get current slot: %v", err)
	}

	currentSlotInDB, err := getCurrentSlotInDB(ctx, coll)
	if err != nil {
		log.Fatalf("failed to get current slot in DB: %v", err)
	}

	if (currentSlot - currentSlotInDB) < 300 {
		// keep currentSlot as is
	} else {
		currentSlot = currentSlotInDB + 300
	}

	for i := currentSlotInDB + 1; i <  currentSlot; i++ {
		url := s.config.Dora.URL + "/api/v1/slot/" + toString(i)
		payload, err := fetchJSON(ctx, url)
		if err != nil {
			log.Fatalf("failed to fetch JSON: %v", err)
		}

		e, ok := payload["data"].(map[string]any)
		if !ok {
			log.Fatalf("invalid payload['data'] shape")
		}

		slot := model.Slot{
			Slot:              i,
			Epoch:             toInt(e["epoch"]),
			Time:              toString(e["time"]),
			Finalized:         toBool(e["finalized"]),
			Scheduled:         toBool(e["scheduled"]),
			Status:            toString(e["status"]),
			Proposer:          toInt(e["proposer"]),
			ProposerName:      toString(e["proposer_name"]),
			AttestationCount:  toInt(e["attestation_count"]),
			DepositCount:      toInt(e["deposit_count"]),
			ExitCount:         toInt(e["exit_count"]),
			ProposerSlashingCount: toInt(e["proposer_slashing_count"]),
			AttesterSlashingCount: toInt(e["attester_slashing_count"]),
			SyncAggregateParticipation: toInt(e["sync_aggregate_participation"]),
			EthTransactionCount: toInt(e["eth_transaction_count"]),
			BlobCount:         toInt(e["blob_count"]),
			WithEthBlock:      toBool(e["with_eth_block"]),
			EthBlockNumber: toInt(e["eth_block_number"]),
			Graffiti:          toString(e["graffiti"]),
			GraffitiText:      toString(e["graffiti_text"]),
			ElExtraData:       toString(e["el_extra_data"]),
			GasUsed:           toInt(e["gas_used"]),
			GasLimit:          toInt(e["gas_limit"]),
			BlockSize:         toInt(e["block_size"]),
			BlockRoot:         toString(e["block_root"]),
			ParentRoot:        toString(e["parent_root"]),
			StateRoot:         toString(e["state_root"]),
			RecvDelay:         toInt(e["recv_delay"]),
			IsMevBlock:        toBool(e["is_mev_block"]),
		}
		_, err = saveDocument(ctx, coll, slot)
		if err != nil {
			log.Fatalf("failed to save document: %v", err)
		}
		// log.Printf("saved document with _id=%v to %s.%s", insertedID, s.config.Mongo.DBName, s.collName)
		arrCount[2]++
	}

	log.Printf("\t\t\t FINISHED FETCHING SLOTS\n\n")
	log.Printf("\t\t Inserted: %v\t",  arrCount[2])
}

func getCurrentSlot(arr []any) (int, error) {
	currentSlot := 0
	for _, item := range arr {
		e, ok := item.(map[string]any)
		if !ok {
			log.Fatalf("slot item is not an object")
		}
		slot := toInt(e["slot"])
		if slot > currentSlot {
			currentSlot = slot
		} else {
			return currentSlot, nil
		}
	}
	log.Println("Current slot", currentSlot)

	return currentSlot, nil
}


func getCurrentSlotInDB(ctx context.Context, coll *mongo.Collection) (int, error) {
	filter := bson.M{"slot": bson.M{"$exists": true}}
	sort := bson.M{"slot": -1}
	opts := options.FindOne().SetSort(sort)

	var slot model.Slot
	err := coll.FindOne(ctx, filter, opts).Decode(&slot)
	if err == mongo.ErrNoDocuments {
		return 0, nil
	}
	return slot.Slot, nil
}