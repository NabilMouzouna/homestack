package auth

import (
	"context"
	"testing"

	"homestack/internal/db"
)

func TestStore_CreateSession_LookupSession(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	// Create user
	hash, _ := HashPassword("pass")
	userID, err := database.CreateUser("alice", hash, true)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	store := NewStore(database)
	token, _, err := store.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if token == "" {
		t.Error("token should be non-empty")
	}
	u, err := store.LookupSession(token)
	if err != nil {
		t.Fatalf("LookupSession: %v", err)
	}
	if u == nil || u.Username != "alice" || !u.IsAdmin {
		t.Errorf("LookupSession: got %+v", u)
	}
}

func TestStore_LookupSession_emptyToken(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	store := NewStore(database)
	u, err := store.LookupSession("")
	if err != nil {
		t.Fatalf("LookupSession: %v", err)
	}
	if u != nil {
		t.Errorf("expected nil user for empty token, got %+v", u)
	}
}

func TestStore_RevokeSession(t *testing.T) {
	dir := t.TempDir()
	database, err := db.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer database.Close()
	userID, _ := database.CreateUser("bob", "hash", false)
	store := NewStore(database)
	token, _, _ := store.CreateSession(userID)
	err = store.RevokeSession(token)
	if err != nil {
		t.Fatalf("RevokeSession: %v", err)
	}
	u, _ := store.LookupSession(token)
	if u != nil {
		t.Errorf("after revoke expected nil user, got %+v", u)
	}
}

func TestWithUser_UserFromContext_IsAdmin(t *testing.T) {
	ctx := context.Background()
	if UserFromContext(ctx) != nil {
		t.Error("empty context should have nil user")
	}
	if IsAdmin(ctx) {
		t.Error("empty context should not be admin")
	}
	u := &User{ID: 1, Username: "admin", IsAdmin: true}
	ctx = WithUser(ctx, u)
	got := UserFromContext(ctx)
	if got != u {
		t.Errorf("UserFromContext: got %+v", got)
	}
	if !IsAdmin(ctx) {
		t.Error("IsAdmin should be true")
	}
	ctx2 := WithUser(context.Background(), &User{ID: 2, Username: "user", IsAdmin: false})
	if IsAdmin(ctx2) {
		t.Error("non-admin user should not be admin")
	}
}
