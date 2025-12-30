DB_PATH ?= ../data/telemetry.db
INGEST_ADDR ?= :7070

run:
	DB_PATH=$(DB_PATH) INGEST_ADDR=$(INGEST_ADDR) go run cmd/ingest/main.go
