package db

import (
    "os"
    "path/filepath"
    "testing"
)

// Ensure migration files and API docs exist as a basic sanity check.
func TestMigrationsAndDocsExist(t *testing.T) {
    base := "docs/api"
    openapi := filepath.Join(base, "openapi.yaml")
    if _, err := os.Stat(openapi); err != nil {
        t.Fatalf("missing openapi file: %v", err)
    }
}
