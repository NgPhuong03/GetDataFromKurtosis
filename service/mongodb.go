package service

import (
	"context"
	"errors"
	"mongo_fetch/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


func connectMongo(ctx context.Context) (*mongo.Client, error) {
	config := config.LoadConfig()
	if config.Mongo.MongoURI == "" {
		return nil, errors.New("MONGO_URI is empty")
	}
	clientOptions := options.Client().ApplyURI(config.Mongo.MongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	return client, nil
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