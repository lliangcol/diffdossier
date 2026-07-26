package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishedSchemasAreValidJSON(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "*.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) < 7 {
		t.Fatalf("found %d schemas, want at least 7", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var document map[string]any
			if err := json.Unmarshal(content, &document); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if document["$schema"] == nil || document["$id"] == nil {
				t.Fatal("published schema requires $schema and $id")
			}
		})
	}
}
