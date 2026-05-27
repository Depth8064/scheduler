package api

import (
    "os"
    "path/filepath"
    "testing"
)

func TestOpenAPIDocumentValidity(t *testing.T) {
    candidates := []string{
        filepath.Join("docs", "api", "openapi.yaml"),
        filepath.Join("..", "..", "docs", "api", "openapi.yaml"),
    }
    var data []byte
    var err error
    for _, c := range candidates {
        data, err = os.ReadFile(c)
        if err == nil {
            break
        }
    }
    if err != nil {
        t.Fatalf("openapi.yaml not found: %v", err)
    }

    text := string(data)
    expected := []string{"/auth/login", "/admin/workstations"}
    for _, p := range expected {
        if !contains(text, p) {
            t.Fatalf("openapi.yaml does not contain expected path: %s", p)
        }
    }
}

func contains(s, sub string) bool {
    return len(s) >= len(sub) && (len(sub) == 0 || (len(s) > 0 && (stringIndex(s, sub) >= 0)))
}

// simple substring index to avoid importing strings in this tiny test
func stringIndex(s, sep string) int {
    n := len(s)
    m := len(sep)
    if m == 0 {
        return 0
    }
    for i := 0; i <= n-m; i++ {
        if s[i:i+m] == sep {
            return i
        }
    }
    return -1
}
