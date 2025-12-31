# Telemetry Lab (Polyglot)

A polyglot practice project to get practice in:
- Go (ingestion gateway)
- C++ (parsing/rules engine)
- Python (analytics + anomaly detection)
- Java/Spring Boot (API + OpenAPI)
- (UI later) React/TypeScript

## Goals
- Practice SDLC: issues, PRs, reviews, ADRs
- Practice CI: tests + basic quality gates
- Practice documentation: UML diagrams + runbook

## Repo layout
- ingest-go/ (Go ingest service)
- engine-cpp/ (planned)
- analytics-py/ (planned)
- api-java/ (planned)
- ui/ (planned)
- data/ (local SQLite DB, gitignored)
- docs/
  - adr/ (architecture decision records)
  - diagrams/ (PlantUML)

## Go Ingest Service
**What it does:** accepts telemetry events and stores them in SQLite.

**Run:**
```bash
cd ingest-go
make run
```
**Database:**
- Default path: `../data/telemetry.db`
- WAL mode creates `telemetry.db-wal` and `telemetry.db-shm` (not committed)

**Docs:**
- Swagger UI: `htttp://localhost:7070/docs`
- OpenAPI YAML: `http://localhost:7070/openapi.yaml`

**Endpoints**
- `GET /healthz`
- `POST /events`
- `POST /events/batch`

**Example**
```bash
curl -s -X POST http://localhost:7070/events \
  -H "Content-Type: application/json" \
  -d '{"source":"app1","ts":"2025-12-30T12:00:00Z","level":"INFO","message":"user login ok","meta":{"userId":"123"}}'
```

## Definition of Done (for PRs)
- Code + tests updated
- Docs updated if behavior/architecture changes
- ADR added/updated if a decision was made
- Diagrams updated if boundaries or flows changed

