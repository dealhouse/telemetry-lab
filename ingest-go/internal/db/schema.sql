PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
PRAGMA busy_timeout=5000;

CREATE TABLE IF NOT EXISTS events (
  id          TEXT PRIMARY KEY,
  source      TEXT NOT NULL,
  ts          TEXT NOT NULL,         -- RFC3339
  level       TEXT NOT NULL,
  message     TEXT NOT NULL,
  meta_json   TEXT,                  -- JSON stored as TEXT
  received_at TEXT NOT NULL          -- RFC3339
);

CREATE INDEX IF NOT EXISTS idx_events_ts     ON events(ts);
CREATE INDEX IF NOT EXISTS idx_events_source ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_level  ON events(level);
