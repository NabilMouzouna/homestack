package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"homestack/internal/db"
	"homestack/internal/db/queries"
)

// CookieName is the HTTP cookie name for the session token.
const CookieName = "homestack_session"

// Store creates and looks up sessions via the database.
type Store struct {
	db *db.DB
}

// NewStore returns a session store that uses the given DB.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// CreateSession generates a token, stores the session, and returns the token and expiry.
// Caller should set a cookie with CookieName and the returned token.
func (s *Store) CreateSession(userID int64) (token string, expiresAt time.Time, err error) {
	token, err = generateToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt = time.Now().UTC().Add(queries.SessionLifetime)
	_, err = s.db.CreateSession(token, userID, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

// LookupSession returns the user for the given token if the session exists and is not expired.
// Returns nil if token is empty, not found, or expired.
func (s *Store) LookupSession(token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	_, u, err := s.db.GetSessionByToken(token)
	if err != nil {
		if err == queries.ErrNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &User{ID: u.ID, Username: u.Username, IsAdmin: u.IsAdmin}, nil
}

// RevokeSession removes the session for the given token.
func (s *Store) RevokeSession(token string) error {
	return s.db.DeleteSession(token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// contextKey is the type for context keys used by auth.
type contextKey int

const (
	contextKeyUser contextKey = iota
)

// WithUser returns a context that carries the current user.
func WithUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, contextKeyUser, u)
}

// UserFromContext returns the current user from the request context, or nil if not logged in.
func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(contextKeyUser).(*User)
	return u
}

// IsAdmin returns true if the context has a user and that user is an admin.
func IsAdmin(ctx context.Context) bool {
	u := UserFromContext(ctx)
	return u != nil && u.IsAdmin
}
