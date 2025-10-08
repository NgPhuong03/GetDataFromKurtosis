package service

import (
	"context"
	"errors"
	"mongo_fetch/config"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
    mongoClient     *mongo.Client
    mongoClientOnce sync.Once
)

// getMongoClient returns a process-wide singleton Mongo client.
// Params: ctx used for initial connect and ping on first call.
// Returns: connected *mongo.Client or error.
// Usage: client, err := getMongoClient(ctx)
func getMongoClient(ctx context.Context) (*mongo.Client, error) {
    var initErr error
    mongoClientOnce.Do(func() {
        cfg := config.LoadConfig()
        if cfg.Mongo.MongoURI == "" {
            initErr = errors.New("MONGO_URI is empty")
            return
        }
        clientOptions := options.Client().ApplyURI(cfg.Mongo.MongoURI)
        c, err := mongo.Connect(ctx, clientOptions)
        if err != nil {
            initErr = err
            return
        }
        if err := c.Ping(ctx, nil); err != nil {
            _ = c.Disconnect(context.Background())
            initErr = err
            return
        }
        mongoClient = c
    })
    if initErr != nil {
        return nil, initErr
    }
    if mongoClient == nil {
        return nil, errors.New("mongo client not initialized")
    }
    return mongoClient, nil
}

// disconnectMongo gracefully closes the singleton client; no-op if not initialized.
// Params: ctx for shutdown deadline.
// Usage: defer disconnectMongo(ctx)
func DisconnectMongo(ctx context.Context) error {
    if mongoClient == nil {
        return nil
    }
    return mongoClient.Disconnect(ctx)
}

// saveDocument inserts the provided document into the specified collection.
// Params: ctx, collection handle, doc to insert.
// Returns: the inserted ID on success.
// Usage: id, err := saveDocument(ctx, coll, bson.M{"k":"v"})
func saveDocument(ctx context.Context, coll *mongo.Collection, doc any) (any, error) {
    res, err := coll.InsertOne(ctx, doc)
    if err != nil {
        return nil, err
    }
    return res.InsertedID, nil
}