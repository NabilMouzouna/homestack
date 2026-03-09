package auth

import (
	"testing"
)

func TestHashPassword_ComparePassword(t *testing.T) {
	plain := "secret123"
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == plain || len(hash) == 0 {
		t.Errorf("hash should not equal plain and be non-empty")
	}
	if err := ComparePassword(hash, plain); err != nil {
		t.Errorf("ComparePassword(same): %v", err)
	}
	if err := ComparePassword(hash, "wrong"); err == nil {
		t.Error("ComparePassword(wrong) should fail")
	}
}

func TestHashPassword_differentSalts(t *testing.T) {
	plain := "same"
	h1, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword 1: %v", err)
	}
	h2, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword 2: %v", err)
	}
	if h1 == h2 {
		t.Error("two hashes of same password should differ (salt)")
	}
	if err := ComparePassword(h1, plain); err != nil {
		t.Errorf("ComparePassword h1: %v", err)
	}
	if err := ComparePassword(h2, plain); err != nil {
		t.Errorf("ComparePassword h2: %v", err)
	}
}
