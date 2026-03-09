package storage

import (
	"path/filepath"
	"testing"
)

func TestGuard_Validate(t *testing.T) {
	root := t.TempDir()
	g, err := NewGuard(root)
	if err != nil {
		t.Fatalf("NewGuard: %v", err)
	}

	tests := []struct {
		name    string
		rel     string
		wantErr bool
	}{
		{"empty", "", false},
		{"dot", ".", false},
		{"subdir", "foo", false},
		{"nested", "a/b/c", false},
		{"escape parent", "..", true},
		{"escape with slash", "../etc/passwd", true},
		{"escape embedded", "foo/../../bar", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := g.Validate(tt.rel)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) err = %v, wantErr %v", tt.rel, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != "" {
				// Result must be under root
				rel, _ := filepath.Rel(g.Root(), got)
				if rel == ".." || len(rel) >= 2 && rel[:2] == ".." {
					t.Errorf("Validate(%q) returned path outside root: %s", tt.rel, got)
				}
			}
		})
	}
}

func TestGuard_Root(t *testing.T) {
	root := t.TempDir()
	g, err := NewGuard(root)
	if err != nil {
		t.Fatal(err)
	}
	if g.Root() != root {
		t.Errorf("Root() = %s, want %s", g.Root(), root)
	}
}

func TestNewGuard_invalidPath(t *testing.T) {
	// NewGuard with empty string still resolves to cwd on most systems; use a path that can't be absoluted if needed
	// On Unix, "" gives cwd. So we test that Root is absolute.
	g, err := NewGuard(".")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(g.Root()) {
		t.Errorf("Root() should be absolute, got %s", g.Root())
	}
}
