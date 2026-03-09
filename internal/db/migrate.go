package db

import (
	"database/sql"
	"embed"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

const schemaVersionTable = "schema_version"

// runMigrations creates schema_version if needed and runs pending migrations.
func runMigrations(conn *sql.DB) error {
	_, err := conn.Exec(`CREATE TABLE IF NOT EXISTS ` + schemaVersionTable + ` (version INTEGER NOT NULL PRIMARY KEY);`)
	if err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	var current int
	err = conn.QueryRow("SELECT COALESCE(MAX(version), 0) FROM " + schemaVersionTable).Scan(&current)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, name := range names {
		verStr := strings.TrimSuffix(name, ".sql")
		parts := strings.SplitN(verStr, "_", 2)
		if len(parts) != 2 {
			continue
		}
		ver, err := strconv.Atoi(parts[0])
		if err != nil || ver <= current {
			continue
		}
		body, err := migrationsFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := conn.Exec(string(body)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}
		if _, err := conn.Exec("INSERT INTO "+schemaVersionTable+" (version) VALUES (?)", ver); err != nil {
			return fmt.Errorf("record schema version %d: %w", ver, err)
		}
		current = ver
	}
	return nil
}
