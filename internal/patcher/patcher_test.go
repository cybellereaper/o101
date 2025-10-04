package patcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cybellereaper/open101/internal/state"
)

type memoryLogger struct {
	logs []string
}

func (l *memoryLogger) Printf(format string, args ...any) {
	l.logs = append(l.logs, fmt.Sprintf(format, args...))
}

func newTestServer(t *testing.T, version string, files map[string][]byte) *httptest.Server {
	t.Helper()

	manifest := map[string]any{
		"version": version,
		"files":   make([]map[string]any, 0, len(files)),
	}

	for name, data := range files {
		sum := sha256.Sum256(data)
		manifest["files"] = append(manifest["files"].([]map[string]any), map[string]any{
			"src":    "files/" + name,
			"dst":    name,
			"size":   len(data),
			"sha256": hex.EncodeToString(sum[:]),
		})
	}

	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/patch-info":
			payload := map[string]any{
				"version":  version,
				"manifest": serverURL(r, "/manifest.json"),
			}
			b, _ := json.Marshal(payload)
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write(manifestBytes)
		default:
			if strings.HasPrefix(r.URL.Path, "/files/") {
				name := strings.TrimPrefix(r.URL.Path, "/files/")
				if data, ok := files[name]; ok {
					w.Write(data)
					return
				}
			}
			http.NotFound(w, r)
		}
	}))

	return server
}

func serverURL(r *http.Request, path string) string {
	return "http://" + r.Host + path
}

func TestPatcherRunDownloadsMissingFiles(t *testing.T) {
	files := map[string][]byte{
		"Bin/app.bin": []byte("hello world"),
	}

	server := newTestServer(t, "1.0.0", files)
	defer server.Close()

	dir := t.TempDir()
	store := &state.Store{Path: filepath.Join(dir, "state.json")}

	logger := &memoryLogger{}
	p, err := New(Config{
		PatchInfoURL: server.URL + "/patch-info",
		InstallDir:   dir,
		StateStore:   store,
		HTTPClient:   server.Client(),
		Logger:       logger,
		Concurrency:  2,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	dest := filepath.Join(dir, "Bin", "app.bin")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != "hello world" {
		t.Fatalf("unexpected file contents %q", data)
	}

	snapshot, err := store.Load(context.Background())
	if err != nil {
		t.Fatalf("Load state: %v", err)
	}

	if snapshot.Version != "1.0.0" {
		t.Fatalf("unexpected version %q", snapshot.Version)
	}
}

func TestPatcherRunSkipsUpToDate(t *testing.T) {
	files := map[string][]byte{
		"Bin/app.bin": []byte("hello world"),
	}
	server := newTestServer(t, "1.0.0", files)
	defer server.Close()

	dir := t.TempDir()
	store := &state.Store{Path: filepath.Join(dir, "state.json")}

	p, err := New(Config{
		PatchInfoURL: server.URL + "/patch-info",
		InstallDir:   dir,
		StateStore:   store,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if err := p.Run(ctx); !errors.Is(err, ErrUpToDate) {
		t.Fatalf("expected ErrUpToDate, got %v", err)
	}
}

func TestPatcherRunRecoversFromCorruption(t *testing.T) {
	files := map[string][]byte{
		"Bin/app.bin": []byte("hello world"),
	}
	server := newTestServer(t, "1.0.0", files)
	defer server.Close()

	dir := t.TempDir()
	store := &state.Store{Path: filepath.Join(dir, "state.json")}

	p, err := New(Config{
		PatchInfoURL: server.URL + "/patch-info",
		InstallDir:   dir,
		StateStore:   store,
		HTTPClient:   server.Client(),
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx := context.Background()
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	dest := filepath.Join(dir, "Bin", "app.bin")
	if err := os.WriteFile(dest, []byte("tampered"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run() after corruption error = %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	if string(data) != "hello world" {
		t.Fatalf("file not restored, got %q", data)
	}
}
