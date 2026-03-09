package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpen(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	if db.Conn() == nil {
		t.Fatal("Conn() is nil")
	}
	path := filepath.Join(dir, "homestack.db")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("database file not created at %s", path)
	}
}

func TestOpen_runsMigrations(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	// Schema version and users table should exist
	var n int
	err = db.Conn().QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&n)
	if err != nil {
		t.Fatalf("schema_version table: %v", err)
	}
	if n < 1 {
		t.Errorf("expected at least one migration recorded, got %d", n)
	}
}

func TestCreateUser_GetUserByUsername_ListUsers_DeleteUser(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	id, err := db.CreateUser("alice", "hash1", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id <= 0 {
		t.Errorf("CreateUser returned id %d", id)
	}
	u, err := db.GetUserByUsername("alice")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if u.Username != "alice" || u.PasswordHash != "hash1" || u.IsAdmin {
		t.Errorf("GetUserByUsername: got %+v", u)
	}
	list, err := db.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list) != 1 || list[0].Username != "alice" {
		t.Errorf("ListUsers: got %v", list)
	}
	err = db.DeleteUser("alice")
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err = db.GetUserByUsername("alice")
	if err == nil {
		t.Fatal("GetUserByUsername after delete should error")
	}
}

func TestCreateUser_duplicate(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	_, err = db.CreateUser("bob", "hash", false)
	if err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}
	_, err = db.CreateUser("bob", "other", false)
	if err == nil {
		t.Fatal("second CreateUser should fail")
	}
}

func TestHasAdmin(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()
	has, err := db.HasAdmin()
	if err != nil {
		t.Fatalf("HasAdmin (empty): %v", err)
	}
	if has {
		t.Error("HasAdmin should be false for empty DB")
	}
	_, err = db.CreateUser("user", "hash", false)
	if err != nil {
		t.Fatalf("CreateUser (non-admin): %v", err)
	}
	has, err = db.HasAdmin()
	if err != nil {
		t.Fatalf("HasAdmin (non-admin): %v", err)
	}
	if has {
		t.Error("HasAdmin should be false with only non-admin users")
	}
	_, err = db.CreateUser("admin", "hash", true)
	if err != nil {
		t.Fatalf("CreateUser (admin): %v", err)
	}
	has, err = db.HasAdmin()
	if err != nil {
		t.Fatalf("HasAdmin (admin): %v", err)
	}
	if !has {
		t.Error("HasAdmin should be true when admin exists")
	}
}
