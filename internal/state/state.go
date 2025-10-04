package state

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// Snapshot captures the persisted patch state for a single installation.
type Snapshot struct {
	Version string              `json:"version"`
	Files   map[string]FileInfo `json:"files"`
}

// FileInfo describes metadata tracked for a single file on disk.
type FileInfo struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Store persists the snapshot to disk in JSON form.
type Store struct {
	Path string
	mu   sync.Mutex
}

// Load reads the current snapshot from disk. Missing files result in an empty snapshot.
func (s *Store) Load(ctx context.Context) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Path == "" {
		return Snapshot{}, errors.New("state: path is required")
	}

	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Snapshot{Files: map[string]FileInfo{}}, nil
		}
		return Snapshot{}, fmt.Errorf("state: read: %w", err)
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("state: decode: %w", err)
	}

	if snapshot.Files == nil {
		snapshot.Files = map[string]FileInfo{}
	}

	return snapshot, nil
}

// Save persists the snapshot to disk atomically.
func (s *Store) Save(ctx context.Context, snapshot Snapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Path == "" {
		return errors.New("state: path is required")
	}

	if snapshot.Files == nil {
		snapshot.Files = map[string]FileInfo{}
	}

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("state: encode: %w", err)
	}

	tmp := s.Path + ".tmp"

	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("state: mkdir: %w", err)
	}

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("state: write tmp: %w", err)
	}

	if err := os.Rename(tmp, s.Path); err != nil {
		return fmt.Errorf("state: rename: %w", err)
	}

	return nil
}
