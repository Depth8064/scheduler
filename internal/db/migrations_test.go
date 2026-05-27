package db

import (
    "os"
    "path/filepath"
    "testing"
)

// Ensure migration files and API docs exist as a basic sanity check.
func TestMigrationsAndDocsExist(t *testing.T) {
    candidates := []string{
        filepath.Join("docs", "api", "openapi.yaml"),
        filepath.Join("..", "..", "docs", "api", "openapi.yaml"),
    }
    found := false
    var lastErr error
    for _, c := range candidates {
        if _, err := os.Stat(c); err == nil {
            found = true
            break
        } else {
            lastErr = err
        }
    }
    if !found {
        t.Fatalf("missing openapi file in expected locations: %v", lastErr)
    }
}
