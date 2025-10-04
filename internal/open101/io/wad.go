package io

import (
	"bytes"
	"compress/flate"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	// ErrInvalidWadHeader indicates the file does not start with the expected KIWAD header.
	ErrInvalidWadHeader = errors.New("wad: invalid header")
	// ErrFileNotFound indicates the requested resource is missing from the archive.
	ErrFileNotFound   = errors.New("wad: file not found")
	errShortWadHeader = errors.New("wad: unexpected end of header")
)

// FileRecord describes a single entry within a WAD archive.
type FileRecord struct {
	Offset         uint32
	Size           uint32
	CompressedSize uint32
	Compressed     bool
	CRC            uint32
	Name           string
}

// Wad represents a parsed KingsIsle WAD archive.
type Wad struct {
	path    string
	records []FileRecord
	index   map[string]FileRecord
}

// OpenWad opens a WAD archive from disk.
func OpenWad(path string) (*Wad, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	records, err := readHeader(f)
	if err != nil {
		return nil, err
	}

	index := make(map[string]FileRecord, len(records))
	for _, record := range records {
		if record.Name == "" {
			continue
		}
		index[record.Name] = record
	}

	return &Wad{path: path, records: records, index: index}, nil
}

// ReadHeader reads WAD header information from an io.Reader.
func ReadHeader(r io.Reader) ([]FileRecord, error) {
	return readHeader(r)
}

func readHeader(r io.Reader) ([]FileRecord, error) {
	header := make([]byte, 5)
	if _, err := io.ReadFull(r, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return nil, errShortWadHeader
		}
		return nil, err
	}
	if !bytes.Equal(header, []byte("KIWAD")) {
		return nil, ErrInvalidWadHeader
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	var numFiles uint32
	if err := binary.Read(r, binary.LittleEndian, &numFiles); err != nil {
		return nil, err
	}
	if version >= 2 {
		if _, err := io.CopyN(io.Discard, r, 1); err != nil {
			return nil, err
		}
	}

	records := make([]FileRecord, 0, numFiles)
	for i := uint32(0); i < numFiles; i++ {
		var record FileRecord
		if err := binary.Read(r, binary.LittleEndian, &record.Offset); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &record.Size); err != nil {
			return nil, err
		}
		if err := binary.Read(r, binary.LittleEndian, &record.CompressedSize); err != nil {
			return nil, err
		}
		var compressed uint8
		if err := binary.Read(r, binary.LittleEndian, &compressed); err != nil {
			return nil, err
		}
		record.Compressed = compressed != 0
		if err := binary.Read(r, binary.LittleEndian, &record.CRC); err != nil {
			return nil, err
		}
		var nameLen int32
		if err := binary.Read(r, binary.LittleEndian, &nameLen); err != nil {
			return nil, err
		}
		if nameLen <= 0 {
			records = append(records, record)
			continue
		}
		nameBytes := make([]byte, nameLen-1)
		if _, err := io.ReadFull(r, nameBytes); err != nil {
			return nil, err
		}
		if terminator, err := readByte(r); err != nil {
			return nil, err
		} else if terminator != 0 {
			return nil, errors.New("wad: expected null terminator")
		}
		record.Name = string(nameBytes)
		records = append(records, record)
	}
	return records, nil
}

func readByte(r io.Reader) (byte, error) {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// OpenFile returns a reader for the specified file inside the archive.
func (w *Wad) OpenFile(name string) (io.ReadCloser, error) {
	record, ok := w.index[name]
	if !ok {
		return nil, ErrFileNotFound
	}

	f, err := os.Open(w.path)
	if err != nil {
		return nil, err
	}

	if _, err := f.Seek(int64(record.Offset), io.SeekStart); err != nil {
		f.Close()
		return nil, err
	}

	chunkSize := record.CompressedSize
	if chunkSize == 0 {
		chunkSize = record.Size
	}
	raw := make([]byte, chunkSize)
	if _, err := io.ReadFull(f, raw); err != nil {
		f.Close()
		return nil, err
	}
	f.Close()

	data := make([]byte, record.Size)
	if record.Compressed {
		if chunkSize < 2 {
			return nil, errors.New("wad: compressed entry too small")
		}
		reader := flate.NewReader(bytes.NewReader(raw[2:chunkSize]))
		defer reader.Close()
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}
	} else {
		copy(data, raw[:record.Size])
	}
	if crc32.ChecksumIEEE(data) != record.CRC {
		return nil, errors.New("wad: crc mismatch")
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// Manager provides cached access to WAD archives in a game installation.
type Manager struct {
	mu      sync.RWMutex
	gameDir string
	dataDir string
	wads    map[string]*Wad
}

// NewManager builds a manager using the supplied game directory. The WAD data directory is assumed to be
// Data/GameData relative to gameDir.
func NewManager(gameDir string) *Manager {
	m := &Manager{wads: make(map[string]*Wad)}
	m.SetGameDir(gameDir)
	return m
}

// SetGameDir reconfigures the manager to use a new Wizard101 installation path.
func (m *Manager) SetGameDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gameDir = dir
	if dir == "" {
		m.dataDir = ""
	} else {
		m.dataDir = filepath.Join(dir, "Data", "GameData")
	}
	m.wads = make(map[string]*Wad)
}

// SetGameDataDir sets the WAD data directory directly.
func (m *Manager) SetGameDataDir(dir string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gameDir = ""
	m.dataDir = dir
	m.wads = make(map[string]*Wad)
}

// GameDir returns the configured game directory.
func (m *Manager) GameDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gameDir
}

// DataDir returns the directory containing the .wad files.
func (m *Manager) DataDir() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.dataDir
}

// OpenFile opens a resource from the underlying archives.
func (m *Manager) OpenFile(name string) (io.ReadCloser, error) {
	wadName, fileName := parseResourceName(name)
	wad, err := m.getWadLocked(wadName)
	if err != nil {
		return nil, err
	}
	return wad.OpenFile(fileName)
}

// GetFileBytes reads the full contents of a resource into memory.
func (m *Manager) GetFileBytes(name string) ([]byte, error) {
	rc, err := m.OpenFile(name)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

// DumpFile writes the specified resource to the current working directory.
func (m *Manager) DumpFile(name string) (string, error) {
	data, err := m.GetFileBytes(name)
	if err != nil {
		return "", err
	}
	_, file := parseResourceName(name)
	if err := os.WriteFile(file, data, 0o644); err != nil {
		return "", err
	}
	return file, nil
}

// AddCustomWad loads an arbitrary WAD file into the manager cache using its base filename as key.
func (m *Manager) AddCustomWad(path string) (*Wad, error) {
	wad, err := OpenWad(path)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	name := filepath.Base(path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	m.wads[name] = wad
	return wad, nil
}

// GetWad exposes a cached wad for testing.
func (m *Manager) GetWad(name string) (*Wad, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	wad, ok := m.wads[name]
	return wad, ok
}

func (m *Manager) getWadLocked(name string) (*Wad, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if wad, ok := m.wads[name]; ok {
		return wad, nil
	}
	if m.dataDir == "" {
		return nil, errors.New("wad: data directory not configured")
	}
	path := filepath.Join(m.dataDir, name+".wad")
	wad, err := OpenWad(path)
	if err != nil {
		return nil, err
	}
	m.wads[name] = wad
	return wad, nil
}

// parseResourceName extracts the wad identifier and inner file path from a resource string.
func parseResourceName(name string) (wadName string, file string) {
	wadName = "Root"
	file = name
	if len(name) > 0 && name[0] == '|' {
		last := strings.LastIndexByte(name, '|')
		if last > 0 {
			wadRaw := name[1:last]
			wadName = strings.ReplaceAll(wadRaw, "|", "-")
			file = name[last+1:]
		}
	}
	file = strings.ReplaceAll(file, "\\", "/")
	return wadName, file
}
