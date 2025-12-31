# ADR 0002: Local dev runtime and storage (No Docker, SQLite, shared DB)

## Status
Accepted

## Context
Local development constraints make Docker impractical for this project. A simple, reliable persistence layer that works consistently across services (Go ingest, later Java API, Python analytics, etc.) and in CI is preferable.

## Decision
- Use SQLite as the local persistence layer for early phases.

- Use WAL mode to improve concurrency characteristics for read-heavy workflows.

- Store the database in a shared repo location: data/telemetry.db so all services can point to the same file.

- Run services directly on the host (no Docker) for local development.

## Consequences
- SQLite supports a limited concurrency model (single-writer), so services should avoid frequent concurrent writes. WAL + sensible timeouts reduce lock pain.

- JSON fields will be stored as TEXT (no Postgres jsonb), which is acceptable for MVP.

- Deployment architecture may evolve later (e.g., Postgres, containers), but the current choice optimizes for speed of iteration and portability.

## Notes / Implementation
- DB files and WAL/SHM artifacts must be gitignored:

    - data/telemetry.db, data/telemetry.db-wal, data/telemetry.db-shm, etc.

- Go ingest defaults to ../data/telemetry.db when run from ingest-go/.
