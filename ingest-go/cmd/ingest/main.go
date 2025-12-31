package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/oklog/ulid/v2"

	_ "modernc.org/sqlite"

	assets "telemetry-lab/ingest-go"
)

type EventIn struct {
	Source  string          `json:"source"`
	TS      string          `json:"ts"`    // RFC3339
	Level   string          `json:"level"` // INFO/WARN/ERROR/DEBUG
	Message string          `json:"message"`
	Meta    json.RawMessage `json:"meta,omitempty"` // object preferred
}

type EventOut struct {
	ID         string `json:"id"`
	ReceivedAt string `json:"receivedAt"`
}

func main() {
	addr := env("INGEST_ADDR", ":7070")
	dbPath := env("DB_PATH", "../data/telemetry.db")

	if strings.HasPrefix(dbPath, "./") || strings.HasPrefix(dbPath, "telemetry.db") {
		log.Fatal("Refusing local DB path. Use ../data/telemetry.db")
	}

	db, err := openSQLite(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(10 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			writeErr(w, http.StatusServiceUnavailable, "db_unhealthy", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	})

	r.Post("/events", func(w http.ResponseWriter, r *http.Request) {
		in, err := decodeEvent(r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		id, receivedAt, err := insertEvent(r.Context(), db, in)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "insert_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, EventOut{ID: id, ReceivedAt: receivedAt})
	})

	r.Post("/events/batch", func(w http.ResponseWriter, r *http.Request) {
		events, err := decodeBatch(r)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		// Validate all first (atomic behavior)
		for i := range events {
			if err := validateEvent(events[i]); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid_event", fmt.Sprintf("index %d: %s", i, err.Error()))
				return
			}
		}

		ids, err := insertBatch(r.Context(), db, events)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "insert_failed", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"count": len(ids), "ids": ids})
	})
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(assets.OpenAPIYAML))
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(
			`<!doctype html>
			<html>
			<head>
			<title>Telemetry Ingest API Docs</title>
			<link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
			</head>
			<body>
			<div id="swagger-ui"></div>
			<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
			<script>
			window.ui = SwaggerUIBundle({ url: "/openapi.yaml", dom_id: "#swagger-ui" });
			</script>
			</body>
			</html>`))
	})

	log.Printf("ingest listening on %s (db=%s)", addr, dbPath)
	log.Fatal(http.ListenAndServe(addr, r))
}

func openSQLite(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// SQLite is happiest if you keep concurrency modest.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	if _, err := db.ExecContext(ctx, assets.SchemaSQL); err != nil {
		return nil, fmt.Errorf("apply schema: %w", err)
	}

	return db, nil
}

func decodeEvent(r *http.Request) (EventIn, error) {
	var in EventIn
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return EventIn{}, err
	}
	if err := validateEvent(in); err != nil {
		return EventIn{}, err
	}
	return in, nil
}

func decodeBatch(r *http.Request) ([]EventIn, error) {
	var arr []EventIn
	r.Body = http.MaxBytesReader(nil, r.Body, 5<<20) // 5MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&arr); err != nil {
		return nil, err
	}
	if len(arr) == 0 {
		return nil, errors.New("batch must not be empty")
	}
	if len(arr) > 1000 {
		return nil, errors.New("batch too large (max 1000)")
	}
	return arr, nil
}

func validateEvent(in EventIn) error {
	in.Source = strings.TrimSpace(in.Source)
	in.Level = strings.TrimSpace(strings.ToUpper(in.Level))
	in.Message = strings.TrimSpace(in.Message)

	if in.Source == "" {
		return errors.New("source is required")
	}
	if len(in.Source) > 64 {
		return errors.New("source too long (max 64)")
	}
	if in.Message == "" {
		return errors.New("message is required")
	}
	if len(in.Message) > 2000 {
		return errors.New("message too long (max 2000)")
	}

	switch in.Level {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return errors.New("level must be one of DEBUG, INFO, WARN, ERROR")
	}

	// Timestamp must be RFC3339 and not wildly in the future
	t, err := time.Parse(time.RFC3339, in.TS)
	if err != nil {
		return errors.New("ts must be RFC3339 (e.g. 2025-12-30T12:00:00Z)")
	}
	if t.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("ts is too far in the future")
	}

	// If meta provided, ensure it's valid JSON object/array/value (we prefer object)
	if len(in.Meta) > 0 {
		var tmp any
		if err := json.Unmarshal(in.Meta, &tmp); err != nil {
			return errors.New("meta must be valid JSON")
		}
	}

	return nil
}

func insertEvent(ctx context.Context, db *sql.DB, in EventIn) (id string, receivedAt string, err error) {
	id = ulid.Make().String()
	receivedAt = time.Now().UTC().Format(time.RFC3339)

	var metaStr any
	if len(in.Meta) > 0 {
		metaStr = string(in.Meta)
	} else {
		metaStr = nil
	}

	_, err = db.ExecContext(ctx,
		`INSERT INTO events (id, source, ts, level, message, meta_json, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, in.Source, in.TS, in.Level, in.Message, metaStr, receivedAt,
	)
	return
}

func insertBatch(ctx context.Context, db *sql.DB, events []EventIn) ([]string, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO events (id, source, ts, level, message, meta_json, received_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	receivedAt := time.Now().UTC().Format(time.RFC3339)
	ids := make([]string, 0, len(events))

	for _, in := range events {
		id := ulid.Make().String()

		var metaStr any
		if len(in.Meta) > 0 {
			metaStr = string(in.Meta)
		} else {
			metaStr = nil
		}

		if _, err := stmt.ExecContext(ctx, id, in.Source, in.TS, in.Level, in.Message, metaStr, receivedAt); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, code string, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	})
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}
