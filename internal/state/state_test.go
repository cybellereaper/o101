package state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	store := &Store{Path: path}
	ctx := context.Background()

	snap, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if snap.Version != "" || len(snap.Files) != 0 {
		t.Fatalf("unexpected snapshot: %#v", snap)
	}

	snap.Version = "1.0.0"
	snap.Files["file"] = FileInfo{Size: 10, SHA256: "hash"}

	if err := store.Save(ctx, snap); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if len(data) == 0 {
		t.Fatalf("expected state file to be written")
	}

	snap2, err := store.Load(ctx)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if snap2.Version != "1.0.0" {
		t.Fatalf("unexpected version %q", snap2.Version)
	}

	if snap2.Files["file"].Size != 10 {
		t.Fatalf("unexpected file info %+v", snap2.Files["file"])
	}
}
