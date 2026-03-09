package queries

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func connWithSessions(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	// Same schema as migrations 001 + 002
	_, err = conn.Exec(`
		CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, is_admin INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (datetime('now')));
		CREATE TABLE sessions (id INTEGER PRIMARY KEY AUTOINCREMENT, token TEXT NOT NULL UNIQUE, user_id INTEGER NOT NULL REFERENCES users(id), expires_at TEXT NOT NULL, created_at TEXT NOT NULL DEFAULT (datetime('now')));
	`)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	return conn
}

func TestCreateSession_GetSessionByToken(t *testing.T) {
	c := connWithSessions(t)
	// Create a user
	res, _ := c.Exec(`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`, "u1", "h1", 0)
	userID, _ := res.LastInsertId()
	expiresAt := time.Now().UTC().Add(SessionLifetime)
	token := "abc123token"
	id, err := CreateSession(c, token, userID, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if id <= 0 {
		t.Errorf("session id = %d", id)
	}
	s, u, err := GetSessionByToken(c, token)
	if err != nil {
		t.Fatalf("GetSessionByToken: %v", err)
	}
	if s.Token != token || s.UserID != userID {
		t.Errorf("session: %+v", s)
	}
	if u.Username != "u1" || u.ID != userID {
		t.Errorf("user: %+v", u)
	}
}

func TestGetSessionByToken_expired(t *testing.T) {
	c := connWithSessions(t)
	res, _ := c.Exec(`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`, "u1", "h1", 0)
	userID, _ := res.LastInsertId()
	// Expiry in the past
	expiresAt := time.Now().UTC().Add(-time.Hour)
	token := "expired"
	_, err := CreateSession(c, token, userID, expiresAt)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	_, _, err = GetSessionByToken(c, token)
	if err != ErrNotFound {
		t.Errorf("expired session should return ErrNotFound, got %v", err)
	}
}

func TestGetSessionByToken_notFound(t *testing.T) {
	c := connWithSessions(t)
	_, _, err := GetSessionByToken(c, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("got %v", err)
	}
}

func TestDeleteSession(t *testing.T) {
	c := connWithSessions(t)
	res, _ := c.Exec(`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`, "u1", "h1", 0)
	userID, _ := res.LastInsertId()
	expiresAt := time.Now().UTC().Add(SessionLifetime)
	token := "todel"
	_, _ = CreateSession(c, token, userID, expiresAt)
	err := DeleteSession(c, token)
	if err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}
	_, _, err = GetSessionByToken(c, token)
	if err != ErrNotFound {
		t.Errorf("after delete: %v", err)
	}
}
