package storage

import (
	"path/filepath"
	"strings"
)

// Guard restricts file operations to the storage root (ADR-002).
type Guard struct {
	root string
}

func NewGuard(root string) (*Guard, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return &Guard{root: abs}, nil
}

func (g *Guard) Root() string {
	return g.root
}

// Validate returns the canonical path under root, or ErrEscape if path escapes.
func (g *Guard) Validate(relativePath string) (string, error) {
	normalized := filepath.Clean(relativePath)
	if normalized == ".." || strings.HasPrefix(normalized, ".."+string(filepath.Separator)) || strings.Contains(normalized, string(filepath.Separator)+"..") {
		return "", ErrEscape
	}
	cleaned := filepath.Clean("/" + relativePath)
	cleaned = strings.TrimPrefix(cleaned, "/")
	joined := filepath.Join(g.root, cleaned)
	abs, err := filepath.Abs(joined)
	if err != nil {
		return "", err
	}
	if !isUnder(abs, g.root) {
		return "", ErrEscape
	}
	return abs, nil
}

func isUnder(path, dir string) bool {
	// path must be exactly dir or under it (no escape via ..)
	return path == dir || (len(path) > len(dir) && path[len(dir)] == filepath.Separator && strings.HasPrefix(path, dir))
}
