package queries

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// User row (id, username, password_hash, is_admin, created_at).
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	IsAdmin      bool
	CreatedAt    time.Time
}

// CreatedAtLayout is SQLite's default datetime format.
const CreatedAtLayout = "2006-01-02 15:04:05"

var ErrNotFound = errors.New("not found")
var ErrExists = errors.New("user already exists")

// CreateUser inserts a user. Returns ErrExists if username already exists.
func CreateUser(conn *sql.DB, username, passwordHash string, isAdmin bool) (int64, error) {
	res, err := conn.Exec(
		`INSERT INTO users (username, password_hash, is_admin) VALUES (?, ?, ?)`,
		username, passwordHash, boolToInt(isAdmin),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return 0, ErrExists
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetUserByUsername returns the user or ErrNotFound.
func GetUserByUsername(conn *sql.DB, username string) (*User, error) {
	var u User
	var isAdmin int
	var createdAt string
	err := conn.QueryRow(
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.IsAdmin = isAdmin != 0
	if t, err := time.Parse(CreatedAtLayout, createdAt); err == nil {
		u.CreatedAt = t
	} else if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		u.CreatedAt = t
	}
	return &u, nil
}

// ListUsers returns all users ordered by username.
func ListUsers(conn *sql.DB) ([]*User, error) {
	rows, err := conn.Query(`SELECT id, username, password_hash, is_admin, created_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*User
	for rows.Next() {
		var u User
		var isAdmin int
		var createdAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &createdAt); err != nil {
			return nil, err
		}
		u.IsAdmin = isAdmin != 0
		if t, err := time.Parse(CreatedAtLayout, createdAt); err == nil {
			u.CreatedAt = t
		} else if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			u.CreatedAt = t
		}
		list = append(list, &u)
	}
	return list, rows.Err()
}

// DeleteUser removes a user by username. Returns ErrNotFound if not found.
func DeleteUser(conn *sql.DB, username string) error {
	res, err := conn.Exec(`DELETE FROM users WHERE username = ?`, username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// HasAdmin reports whether at least one admin user exists.
func HasAdmin(conn *sql.DB) (bool, error) {
	var dummy int
	err := conn.QueryRow(`SELECT 1 FROM users WHERE is_admin = 1 LIMIT 1`).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// isUniqueViolation returns true for SQLite unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "UNIQUE") && strings.Contains(s, "constraint")
}
