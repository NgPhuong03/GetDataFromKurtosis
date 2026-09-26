package service

import (
	"context"
	"errors"
	"log"
	"mongo_fetch/config"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoClient *mongo.Client
	mongoMu     sync.Mutex
)

// getMongoClient returns a process-wide singleton Mongo client.
// Params: ctx used for initial connect and ping on first call.
// Returns: connected *mongo.Client or error.
// Usage: client, err := getMongoClient(ctx)
func getMongoClient(ctx context.Context) (*mongo.Client, error) {
	mongoMu.Lock()
	defer mongoMu.Unlock()
	if mongoClient != nil {
		return mongoClient, nil
	}

	cfg := config.LoadConfig()
	if cfg.Mongo.MongoURI == "" {
		return nil, errors.New("MONGO_URI is empty")
	}
	c, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.Mongo.MongoURI))
	if err != nil {
		return nil, err
	}
	if err := c.Ping(ctx, nil); err != nil {
		_ = c.Disconnect(context.Background())
		return nil, err
	}
	mongoClient = c
	return mongoClient, nil
}

// disconnectMongo gracefully closes the singleton client; no-op if not initialized.
// Params: ctx for shutdown deadline.
// Usage: defer disconnectMongo(ctx)
func DisconnectMongo(ctx context.Context) error {
	mongoMu.Lock()
	defer mongoMu.Unlock()
	if mongoClient == nil {
		return nil
	}
	err := mongoClient.Disconnect(ctx)
	mongoClient = nil
	return err
}

func upsertDocument(ctx context.Context, coll *mongo.Collection, filter any, doc any) (bool, error) {
	res, err := coll.UpdateOne(ctx, filter, bson.M{"$set": doc}, options.Update().SetUpsert(true))
	if err != nil {
		return false, err
	}
	return res.UpsertedCount > 0, nil
}

func ensureIndexes(ctx context.Context, db *mongo.Database) {
	specs := []struct {
		coll string
		name string
		keys bson.D
	}{
		{"slots", "model_slot", bson.D{{Key: "model", Value: 1}, {Key: "slot", Value: 1}}},
		{"epochs", "model_epoch", bson.D{{Key: "model", Value: 1}, {Key: "epoch", Value: 1}}},
		{"validators", "model_pubkey_epoch", bson.D{{Key: "model", Value: 1}, {Key: "public_key", Value: 1}, {Key: "epoch", Value: 1}}},
	}
	for _, spec := range specs {
		_, err := db.Collection(spec.coll).Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    spec.keys,
			Options: options.Index().SetUnique(true).SetName(spec.name),
		})
		if err != nil {
			log.Printf("index %s: %v", spec.name, err)
		}
	}
}
