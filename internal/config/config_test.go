package config

import (
	"testing"
)

func TestLoad_nilConfig(t *testing.T) {
	cfg, err := Load(nil)
	if err != nil {
		t.Fatalf("Load(nil): %v", err)
	}
	if cfg == nil {
		t.Fatal("Load(nil) should return a non-nil config")
	}
	if cfg.Port != 0 {
		t.Errorf("default Port want 0, got %d", cfg.Port)
	}
}

func TestLoad_preservesValues(t *testing.T) {
	in := &Config{AppDataDir: "data", Port: 9000, MaxUsers: 10}
	out, err := Load(in)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if out != in {
		t.Error("Load should return the same config pointer when non-nil")
	}
	if out.AppDataDir != "data" || out.Port != 9000 || out.MaxUsers != 10 {
		t.Errorf("config values changed: %+v", out)
	}
}
