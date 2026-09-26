package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"mongo_fetch/config"
	"mongo_fetch/service"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		usage()
		os.Exit(2)
	}

	cfg := config.LoadConfig()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var err error
	switch os.Args[1] {
	case "fetch":
		if err = cfg.RequireFetch(); err != nil {
			log.Fatal(err)
		}
		err = service.Run(ctx, cfg)
	case "export":
		err = service.ExportDatabase(ctx, cfg)
	default:
		usage()
		os.Exit(2)
	}

	disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = service.DisconnectMongo(disconnectCtx)

	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  go run . fetch     Collect epochs, validators, and slots into the MongoDB database named in config.yaml
  go run . export    Write proposer CSV files from that same database
`)
}
