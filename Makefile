.PHONY: help fetch export test build clean

# Optional overrides. Empty values are ignored by the program.
#   make fetch MODEL=desw DORA_URL=http://127.0.0.1:32778
#   make fetch MODEL=desw TARGET_EPOCHS=5
MODEL ?=
DORA_URL ?=
TARGET_EPOCHS ?=
SLOTS_PER_EPOCH ?=
OUTPUT_DIR ?=

help:
	@echo "make fetch     Collect epochs, validators, and slots into mongo.dbName"
	@echo "make export    Write CSV files from the database named in config.yaml"
	@echo "make test"
	@echo "make build"

fetch:
	MODEL='$(MODEL)' DORA_URL='$(DORA_URL)' TARGET_EPOCHS='$(TARGET_EPOCHS)' SLOTS_PER_EPOCH='$(SLOTS_PER_EPOCH)' OUTPUT_DIR='$(OUTPUT_DIR)' go run . fetch

export:
	OUTPUT_DIR='$(OUTPUT_DIR)' TARGET_EPOCHS='$(TARGET_EPOCHS)' SLOTS_PER_EPOCH='$(SLOTS_PER_EPOCH)' go run . export

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/getdata .

clean:
	rm -rf bin
