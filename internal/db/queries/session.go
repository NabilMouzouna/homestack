package queries

import (
	"database/sql"
	"errors"
	"time"
)

// Session row (id, token, user_id, expires_at, created_at).
type Session struct {
	ID        int64
	Token     string
	UserID    int64
	ExpiresAt time.Time
	CreatedAt time.Time
}

// SessionLifetime is the default duration a session stays valid.
const SessionLifetime = 24 * time.Hour

// CreateSession inserts a session. Caller must provide a unique token and expires_at.
func CreateSession(conn *sql.DB, token string, userID int64, expiresAt time.Time) (int64, error) {
	res, err := conn.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, formatTime(expiresAt),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetSessionByToken returns the session and joined user if token is valid and not expired.
// Returns ErrNotFound if token missing or expired.
func GetSessionByToken(conn *sql.DB, token string) (*Session, *User, error) {
	var s Session
	var u User
	var isAdmin int
	var expiresAt, createdAt string
	var uCreatedAt string
	err := conn.QueryRow(`
		SELECT s.id, s.token, s.user_id, s.expires_at, s.created_at,
		       u.id, u.username, u.password_hash, u.is_admin, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > datetime('now')
	`, token).Scan(
		&s.ID, &s.Token, &s.UserID, &expiresAt, &createdAt,
		&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &uCreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrNotFound
		}
		return nil, nil, err
	}
	s.ExpiresAt, _ = time.Parse(CreatedAtLayout, expiresAt)
	s.CreatedAt, _ = time.Parse(CreatedAtLayout, createdAt)
	u.IsAdmin = isAdmin != 0
	u.CreatedAt, _ = time.Parse(CreatedAtLayout, uCreatedAt)
	return &s, &u, nil
}

// DeleteSession removes a session by token.
func DeleteSession(conn *sql.DB, token string) error {
	res, err := conn.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// CleanupExpiredSessions deletes sessions that have passed expires_at.
func CleanupExpiredSessions(conn *sql.DB) (int64, error) {
	res, err := conn.Exec(`DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}
