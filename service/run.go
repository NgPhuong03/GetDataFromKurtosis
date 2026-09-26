package service

import (
	"context"
	"log"
	"mongo_fetch/config"
	"time"
)

func Run(ctx context.Context, cfg *config.Config) error {
	client, err := getMongoClient(ctx)
	if err != nil {
		return err
	}
	ensureIndexes(ctx, client.Database(cfg.Mongo.DBName))

	epochs := NewEpochService(cfg)
	validators := NewValidatorsService(cfg)
	slots := NewSlotService(cfg)

	log.Printf("collecting model=%s target_epochs=%d slots_per_epoch=%d dora=%s",
		cfg.Model, cfg.TargetEpochs, cfg.SlotsPerEpoch, cfg.Dora.URL)

	for {
		if err := epochs.Sync(ctx); err != nil {
			log.Printf("epochs: %v", err)
		}
		if err := validators.Sync(ctx); err != nil {
			log.Printf("validators: %v", err)
		}
		if err := slots.Sync(ctx); err != nil {
			log.Printf("slots: %v", err)
		}

		status, err := Progress(ctx, cfg)
		if err != nil {
			log.Printf("progress: %v", err)
		} else {
			log.Printf("database=%s %s", cfg.Mongo.DBName, statusLine(status))
			if status.Complete {
				log.Printf("target reached in database %s; run: go run . export", cfg.Mongo.DBName)
				return nil
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(cfg.TimeLoop) * time.Second):
		}
	}
}
