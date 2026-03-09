package queries

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func conn(t *testing.T) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	_, _ = conn.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, is_admin INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT (datetime('now')))`)
	return conn
}

func TestCreateUser_GetUserByUsername(t *testing.T) {
	c := conn(t)
	id, err := CreateUser(c, "u1", "h1", false)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if id <= 0 {
		t.Errorf("id = %d", id)
	}
	u, err := GetUserByUsername(c, "u1")
	if err != nil {
		t.Fatalf("GetUserByUsername: %v", err)
	}
	if u.Username != "u1" || u.PasswordHash != "h1" || u.IsAdmin {
		t.Errorf("got %+v", u)
	}
}

func TestGetUserByUsername_notFound(t *testing.T) {
	c := conn(t)
	_, err := GetUserByUsername(c, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("got err %v", err)
	}
}

func TestCreateUser_duplicate(t *testing.T) {
	c := conn(t)
	_, err := CreateUser(c, "dup", "h", false)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err = CreateUser(c, "dup", "h2", false)
	if err != ErrExists {
		t.Errorf("got err %v", err)
	}
}

func TestListUsers(t *testing.T) {
	c := conn(t)
	_, _ = CreateUser(c, "a", "1", false)
	_, _ = CreateUser(c, "b", "2", true)
	list, err := ListUsers(c)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("len(list)=%d", len(list))
	}
	if list[0].Username != "a" || list[1].Username != "b" {
		t.Errorf("order: %v", list)
	}
	if !list[1].IsAdmin {
		t.Error("second user should be admin")
	}
}

func TestDeleteUser(t *testing.T) {
	c := conn(t)
	_, _ = CreateUser(c, "del", "h", false)
	err := DeleteUser(c, "del")
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err = GetUserByUsername(c, "del")
	if err != ErrNotFound {
		t.Errorf("after delete: %v", err)
	}
}

func TestDeleteUser_notFound(t *testing.T) {
	c := conn(t)
	err := DeleteUser(c, "nobody")
	if err != ErrNotFound {
		t.Errorf("got %v", err)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	if isUniqueViolation(nil) {
		t.Error("nil should be false")
	}
	if !isUniqueViolation(errors.New("UNIQUE constraint failed: users.username")) {
		t.Error("UNIQUE constraint should be true")
	}
	if isUniqueViolation(errors.New("other error")) {
		t.Error("other error should be false")
	}
}

func TestHasAdmin(t *testing.T) {
	c := conn(t)
	has, err := HasAdmin(c)
	if err != nil {
		t.Fatalf("HasAdmin (empty): %v", err)
	}
	if has {
		t.Error("HasAdmin should be false for empty users table")
	}
	_, err = CreateUser(c, "user", "h", false)
	if err != nil {
		t.Fatalf("CreateUser (non-admin): %v", err)
	}
	has, err = HasAdmin(c)
	if err != nil {
		t.Fatalf("HasAdmin (non-admin): %v", err)
	}
	if has {
		t.Error("HasAdmin should be false when only non-admin users exist")
	}
	_, err = CreateUser(c, "admin", "h", true)
	if err != nil {
		t.Fatalf("CreateUser (admin): %v", err)
	}
	has, err = HasAdmin(c)
	if err != nil {
		t.Fatalf("HasAdmin (admin): %v", err)
	}
	if !has {
		t.Error("HasAdmin should be true when an admin exists")
	}
}
