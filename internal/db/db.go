package db

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"homestack/internal/db/queries"
)

// DB provides access to SQLite (users, sessions, config).
type DB struct {
	conn *sql.DB
}

// Open creates or opens the SQLite DB in appDataDir and runs migrations.
// The database file is homestack.db inside appDataDir.
func Open(appDataDir string) (*DB, error) {
	dbPath := filepath.Join(appDataDir, "homestack.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", dbPath, err)
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := runMigrations(conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrations: %w", err)
	}
	return &DB{conn: conn}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

// Conn returns the underlying *sql.DB for use by query packages.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// ErrNotFound is returned when a user or session is not found.
var ErrNotFound = queries.ErrNotFound

// ErrExists is returned when creating a user that already exists.
var ErrExists = queries.ErrExists

// CreateUser inserts a user. Returns queries.ErrExists if username already exists.
func (db *DB) CreateUser(username, passwordHash string, isAdmin bool) (int64, error) {
	return queries.CreateUser(db.conn, username, passwordHash, isAdmin)
}

// GetUserByUsername returns the user or queries.ErrNotFound.
func (db *DB) GetUserByUsername(username string) (*queries.User, error) {
	return queries.GetUserByUsername(db.conn, username)
}

// ListUsers returns all users ordered by username.
func (db *DB) ListUsers() ([]*queries.User, error) {
	return queries.ListUsers(db.conn)
}

// DeleteUser removes a user by username. Returns queries.ErrNotFound if not found.
func (db *DB) DeleteUser(username string) error {
	return queries.DeleteUser(db.conn, username)
}

// HasAdmin reports whether at least one admin user exists.
func (db *DB) HasAdmin() (bool, error) {
	return queries.HasAdmin(db.conn)
}

// CreateSession inserts a session; returns the new session id.
func (db *DB) CreateSession(token string, userID int64, expiresAt time.Time) (int64, error) {
	return queries.CreateSession(db.conn, token, userID, expiresAt)
}

// GetSessionByToken returns the session and user if token is valid and not expired.
func (db *DB) GetSessionByToken(token string) (*queries.Session, *queries.User, error) {
	return queries.GetSessionByToken(db.conn, token)
}

// DeleteSession removes a session by token.
func (db *DB) DeleteSession(token string) error {
	return queries.DeleteSession(db.conn, token)
}
