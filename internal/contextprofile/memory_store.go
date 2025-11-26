package contextprofile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var (
	errMemoryNotFound = errors.New("memory not found")
	errPinLimit       = errors.New("pin limit reached")
)

type memoryEntry struct {
	ID               string
	Content          string
	Summary          string
	Placeholder      string
	OriginalMessages []byte // JSON-encoded []state.Message for full restoration
	CreatedAt        time.Time
	LastAccess       time.Time
	Pinned           bool
}

type memoryStore struct {
	db   *sql.DB
	path string
}

func newMemoryStore(path string) (*memoryStore, error) {
	if path == "" {
		return nil, errors.New("memory store path must be set")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("prepare memory store dir: %w", err)
	}
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_pragma=journal_mode(WAL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open memory store: %w", err)
	}

	// Check for database corruption and attempt recovery
	recoveredDb, err := checkAndRecoverDatabase(db, path)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("database recovery failed: %w", err)
	}
	if recoveredDb != nil {
		// Database was recreated, use the new connection
		db = recoveredDb
	}

	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS memories (
	id TEXT PRIMARY KEY,
	content TEXT NOT NULL,
	summary TEXT NOT NULL,
	placeholder TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL,
	last_access TIMESTAMP NOT NULL,
	pinned INTEGER NOT NULL DEFAULT 0,
	original_messages TEXT
)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init memory schema: %w", err)
	}

	// Migration: Add original_messages column if it doesn't exist
	var hasColumn int
	err = db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM pragma_table_info('memories') WHERE name='original_messages'`).Scan(&hasColumn)
	if err == nil && hasColumn == 0 {
		// Column doesn't exist, add it
		if _, err = db.ExecContext(context.Background(),
			`ALTER TABLE memories ADD COLUMN original_messages TEXT`); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate memory schema: %w", err)
		}
	}

	// Create compaction_events table
	if _, err := db.ExecContext(context.Background(), `
CREATE TABLE IF NOT EXISTS compaction_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	timestamp TIMESTAMP NOT NULL,
	chars_before INTEGER NOT NULL,
	chars_after INTEGER NOT NULL,
	messages_compacted INTEGER NOT NULL,
	messages_considered INTEGER NOT NULL,
	duration_ms INTEGER NOT NULL
)`); err != nil {
		db.Close()
		return nil, fmt.Errorf("init compaction_events schema: %w", err)
	}

	return &memoryStore{db: db, path: path}, nil
}

// checkAndRecoverDatabase detects and recovers from database corruption
// Returns a new *sql.DB if database was recreated, nil if existing db is fine
func checkAndRecoverDatabase(db *sql.DB, path string) (*sql.DB, error) {
	// Check if database file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// New database, nothing to recover - caller will create schema
			return nil, nil
		}
		return nil, err
	}

	isCorrupted := false

	// Check 1: Empty file
	if info.Size() == 0 {
		isCorrupted = true
		fmt.Printf("WARNING: Memory database is empty (0 bytes), attempting recovery\n")
	}

	// Check 2: Verify schema exists
	if !isCorrupted {
		var tableName string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='memories'").Scan(&tableName)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No schema - database is effectively empty
				isCorrupted = true
				fmt.Printf("WARNING: Memory database has no schema, attempting recovery\n")
			} else {
				// Other error (file might be corrupt)
				isCorrupted = true
				fmt.Printf("WARNING: Memory database schema check failed (%v), attempting recovery\n", err)
			}
		}
	}

	if !isCorrupted {
		return nil, nil // Database is healthy, use existing connection
	}

	// Attempt recovery: Try WAL checkpoint first
	fmt.Printf("Attempting WAL checkpoint recovery...\n")
	if _, err := db.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err == nil {
		// Check if recovery worked
		var tableName string
		if err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='memories'").Scan(&tableName); err == nil {
			fmt.Printf("SUCCESS: Database recovered from WAL checkpoint\n")
			return nil, nil // Use existing connection, it's recovered
		}
	}

	// WAL recovery failed - need to recreate database
	fmt.Printf("WAL recovery failed, recreating database from scratch\n")

	// Close the database connection
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("close corrupted database: %w", err)
	}

	// Delete corrupted database files
	os.Remove(path)
	os.Remove(path + "-wal")
	os.Remove(path + "-shm")

	// Reopen with a fresh connection
	// Note: The caller will handle schema creation after this function returns
	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_pragma=journal_mode(WAL)", path)
	newDb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("reopen database after corruption: %w", err)
	}

	fmt.Printf("Database recreated successfully\n")
	return newDb, nil // Return new connection
}

func (s *memoryStore) Put(entry *memoryEntry) error {
	if entry == nil {
		return nil
	}
	_, err := s.db.ExecContext(context.Background(), `
INSERT INTO memories (id, content, summary, placeholder, original_messages, created_at, last_access, pinned)
VALUES(?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
	content=excluded.content,
	summary=excluded.summary,
	placeholder=excluded.placeholder,
	original_messages=excluded.original_messages,
	created_at=excluded.created_at,
	last_access=excluded.last_access,
	pinned=excluded.pinned
`, entry.ID, entry.Content, entry.Summary, entry.Placeholder, entry.OriginalMessages, entry.CreatedAt, entry.LastAccess, boolToInt(entry.Pinned))
	return err
}

func (s *memoryStore) Access(id string, mutate func(*memoryEntry)) (*memoryEntry, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	entry, err := fetchEntry(tx, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if mutate != nil {
		mutate(entry)
		if err := saveEntry(tx, entry); err != nil {
			tx.Rollback()
			return nil, err
		}
	} else {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return entry, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *memoryStore) Pin(id string, pin bool, maxPins int) (*memoryEntry, error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	entry, err := fetchEntry(tx, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if pin && !entry.Pinned {
		var count int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM memories WHERE pinned=1`).Scan(&count); err != nil {
			tx.Rollback()
			return nil, err
		}
		if count >= maxPins {
			tx.Rollback()
			return nil, errPinLimit
		}
		entry.Pinned = true
	} else if !pin {
		entry.Pinned = false
	}
	entry.LastAccess = time.Now()
	if err := saveEntry(tx, entry); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return entry, nil
}

func (s *memoryStore) PinnedCount() int {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM memories WHERE pinned=1`).Scan(&count); err != nil {
		return 0
	}
	return count
}

func (s *memoryStore) Stats(limit int) (int, int, []memoryEntry, error) {
	if limit <= 0 {
		limit = 5
	}
	var total, pinned int
	if err := s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(pinned),0) FROM memories`).Scan(&total, &pinned); err != nil {
		return 0, 0, nil, err
	}
	rows, err := s.db.Query(`
SELECT id, content, summary, placeholder, original_messages, created_at, last_access, pinned
FROM memories
ORDER BY last_access DESC
LIMIT ?`, limit)
	if err != nil {
		return 0, 0, nil, err
	}
	defer rows.Close()
	var entries []memoryEntry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return 0, 0, nil, err
		}
		entries = append(entries, *entry)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, nil, err
	}
	return total, pinned, entries, nil
}

func (s *memoryStore) Path() string {
	return s.path
}

func (s *memoryStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func fetchEntry(execer interface {
	QueryRow(string, ...any) *sql.Row
}, id string) (*memoryEntry, error) {
	row := execer.QueryRow(`SELECT id, content, summary, placeholder, original_messages, created_at, last_access, pinned FROM memories WHERE id=?`, id)
	return scanEntry(row)
}

func saveEntry(exec sqlExecutor, entry *memoryEntry) error {
	_, err := exec.Exec(`UPDATE memories SET content=?, summary=?, placeholder=?, original_messages=?, created_at=?, last_access=?, pinned=? WHERE id=?`,
		entry.Content, entry.Summary, entry.Placeholder, entry.OriginalMessages, entry.CreatedAt, entry.LastAccess, boolToInt(entry.Pinned), entry.ID)
	return err
}

type sqlExecutor interface {
	Exec(string, ...any) (sql.Result, error)
}

func scanEntry(scanner interface {
	Scan(dest ...any) error
}) (*memoryEntry, error) {
	var entry memoryEntry
	var created, access time.Time
	var pinned int
	var originalMessages sql.NullString
	if err := scanner.Scan(&entry.ID, &entry.Content, &entry.Summary, &entry.Placeholder, &originalMessages, &created, &access, &pinned); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errMemoryNotFound
		}
		return nil, err
	}
	entry.CreatedAt = created
	entry.LastAccess = access
	entry.Pinned = pinned == 1
	if originalMessages.Valid && originalMessages.String != "" {
		entry.OriginalMessages = []byte(originalMessages.String)
	}
	return &entry, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// SaveCompactionEvent persists a compaction event to the database
func (s *memoryStore) SaveCompactionEvent(event CompactionEvent) error {
	_, err := s.db.ExecContext(context.Background(), `
INSERT INTO compaction_events (timestamp, chars_before, chars_after, messages_compacted, messages_considered, duration_ms)
VALUES (?, ?, ?, ?, ?, ?)`,
		event.Timestamp, event.CharsBefore, event.CharsAfter, event.MessagesCompacted, event.MessagesConsidered, event.DurationMs)
	return err
}

// LoadCompactionEvents loads all compaction events from the database
func (s *memoryStore) LoadCompactionEvents() ([]CompactionEvent, error) {
	rows, err := s.db.QueryContext(context.Background(), `
SELECT timestamp, chars_before, chars_after, messages_compacted, messages_considered, duration_ms
FROM compaction_events
ORDER BY timestamp DESC
LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CompactionEvent
	for rows.Next() {
		var event CompactionEvent
		if err := rows.Scan(&event.Timestamp, &event.CharsBefore, &event.CharsAfter, &event.MessagesCompacted, &event.MessagesConsidered, &event.DurationMs); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
