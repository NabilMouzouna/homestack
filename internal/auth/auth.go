package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// DefaultCost is the bcrypt cost used for hashing (10).
const DefaultCost = bcrypt.DefaultCost

// User represents a stored user. Password hashing and sessions per ADR-003.
type User struct {
	ID       int64
	Username string
	IsAdmin  bool
}

// HashPassword returns a bcrypt hash of the plaintext password.
// Use this when creating a user; store the result in password_hash.
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// ComparePassword returns nil if plain matches the hashed password.
// Use this on login to validate credentials.
func ComparePassword(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
