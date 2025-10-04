package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Manifest describes the files required to bring an installation to a specific version.
type Manifest struct {
	Version string `json:"version"`
	Files   []File `json:"files"`
}

// File represents a single file download entry within a manifest.
type File struct {
	Source string      `json:"src"`
	Target string      `json:"dst"`
	Size   int64       `json:"size"`
	SHA256 string      `json:"sha256"`
	Mode   fs.FileMode `json:"mode,omitempty"`
}

// Parse validates and converts raw JSON manifest bytes into a strongly typed Manifest.
func Parse(data []byte) (Manifest, error) {
	var raw struct {
		Version string            `json:"version"`
		Files   []json.RawMessage `json:"files"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return Manifest{}, fmt.Errorf("manifest: decode: %w", err)
	}

	if strings.TrimSpace(raw.Version) == "" {
		return Manifest{}, errors.New("manifest: version is required")
	}

	if len(raw.Files) == 0 {
		return Manifest{}, errors.New("manifest: files list is empty")
	}

	result := Manifest{Version: raw.Version, Files: make([]File, 0, len(raw.Files))}

	for i, entry := range raw.Files {
		file, err := decodeFile(entry)
		if err != nil {
			return Manifest{}, fmt.Errorf("manifest: files[%d]: %w", i, err)
		}
		result.Files = append(result.Files, file)
	}

	return result, nil
}

func decodeFile(data json.RawMessage) (File, error) {
	var tmp struct {
		Source string  `json:"src"`
		Target string  `json:"dst"`
		Size   *int64  `json:"size"`
		SHA256 string  `json:"sha256"`
		Mode   *string `json:"mode"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return File{}, fmt.Errorf("decode: %w", err)
	}

	if strings.TrimSpace(tmp.Source) == "" {
		return File{}, errors.New("decode: src is required")
	}

	if path.IsAbs(tmp.Source) {
		return File{}, errors.New("decode: src must be relative")
	}

	if strings.TrimSpace(tmp.Target) == "" {
		return File{}, errors.New("decode: dst is required")
	}

	if path.IsAbs(tmp.Target) {
		return File{}, errors.New("decode: dst must be relative")
	}

	if tmp.Size == nil || *tmp.Size <= 0 {
		return File{}, errors.New("decode: size must be positive")
	}

	if len(tmp.SHA256) != 64 {
		return File{}, errors.New("decode: sha256 must be 64 hex characters")
	}

	if strings.Contains(tmp.Target, "\\") {
		return File{}, errors.New("decode: dst must use forward slashes")
	}

	file := File{
		Source: path.Clean(tmp.Source),
		Target: path.Clean(tmp.Target),
		Size:   *tmp.Size,
		SHA256: strings.ToLower(tmp.SHA256),
		Mode:   0,
	}

	if tmp.Mode != nil && strings.TrimSpace(*tmp.Mode) != "" {
		mode, err := parseFileMode(*tmp.Mode)
		if err != nil {
			return File{}, fmt.Errorf("decode: mode: %w", err)
		}
		file.Mode = mode
	}

	return file, nil
}

func parseFileMode(input string) (fs.FileMode, error) {
	input = strings.TrimSpace(input)
	if len(input) == 0 {
		return 0, errors.New("empty mode")
	}

	var value uint32
	for _, r := range input {
		if r < '0' || r > '7' {
			return 0, fmt.Errorf("invalid digit %q", r)
		}
		value = value*8 + uint32(r-'0')
	}

	return fs.FileMode(value), nil
}
