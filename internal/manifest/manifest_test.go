package manifest

import (
	"encoding/json"
	"testing"
)

func TestParseValidManifest(t *testing.T) {
	payload := map[string]any{
		"version": "1.2.3",
		"files": []map[string]any{
			{
				"src":    "assets/app.bin",
				"dst":    "Bin/app.bin",
				"size":   42,
				"sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				"mode":   "0644",
			},
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	manifest, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if manifest.Version != "1.2.3" {
		t.Fatalf("unexpected version %q", manifest.Version)
	}

	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(manifest.Files))
	}

	entry := manifest.Files[0]
	if entry.Source != "assets/app.bin" || entry.Target != "Bin/app.bin" {
		t.Fatalf("unexpected entry: %#v", entry)
	}

	if entry.Mode != 0o644 {
		t.Fatalf("unexpected mode %v", entry.Mode)
	}
}

func TestParseRejectsInvalidEntries(t *testing.T) {
	payload := map[string]any{
		"version": "1.0.0",
		"files": []map[string]any{
			{"src": "", "dst": "file", "size": 1, "sha256": ""},
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if _, err := Parse(raw); err == nil {
		t.Fatalf("expected error for invalid manifest")
	}
}
