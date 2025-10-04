package io

import (
	"bytes"
	"compress/zlib"
	"errors"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type wadEntry struct {
	name       string
	data       []byte
	compressed bool
}

func buildTestWad(t *testing.T, entries []wadEntry) []byte {
	t.Helper()
	version := uint32(2)
	headerSize := 5 + 4 + 4 + 1
	for _, entry := range entries {
		headerSize += 4 + 4 + 4 + 1 + 4 + 4
		headerSize += len(entry.name) + 1
	}

	buf := bytes.NewBuffer(make([]byte, 0, headerSize))
	buf.WriteString("KIWAD")
	writeUint32(buf, version)
	writeUint32(buf, uint32(len(entries)))
	buf.WriteByte(0)

	dataSection := bytes.NewBuffer(nil)
	offset := uint32(headerSize)
	for _, entry := range entries {
		content := entry.data
		compressedData := content
		compressedSize := uint32(len(content))
		compressedFlag := byte(0)
		if entry.compressed {
			compressedData = compressZlib(t, content)
			compressedSize = uint32(len(compressedData))
			compressedFlag = 1
		}
		writeUint32(buf, offset)
		writeUint32(buf, uint32(len(content)))
		writeUint32(buf, compressedSize)
		buf.WriteByte(compressedFlag)
		writeUint32(buf, crc32.ChecksumIEEE(content))
		writeUint32(buf, uint32(len(entry.name)+1))
		buf.WriteString(entry.name)
		buf.WriteByte(0)

		dataSection.Write(compressedData)
		offset += compressedSize
	}

	buf.Write(dataSection.Bytes())
	return buf.Bytes()
}

func writeUint32(w io.Writer, v uint32) {
	var b [4]byte
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	w.Write(b[:])
}

func compressZlib(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zlib.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		t.Fatalf("compress write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("compress close: %v", err)
	}
	return buf.Bytes()
}

func TestReadHeader(t *testing.T) {
	entries := []wadEntry{{name: "foo/bar.bin", data: []byte("hello"), compressed: false}}
	wadBytes := buildTestWad(t, entries)
	records, err := ReadHeader(bytes.NewReader(wadBytes))
	if err != nil {
		t.Fatalf("ReadHeader error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	rec := records[0]
	if rec.Name != "foo/bar.bin" || rec.Size != 5 || rec.Offset == 0 {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestWadOpenFile(t *testing.T) {
	entries := []wadEntry{
		{name: "plain.txt", data: []byte("plain"), compressed: false},
		{name: "compressed.bin", data: []byte("compress me"), compressed: true},
	}
	wadBytes := buildTestWad(t, entries)
	tmpDir := t.TempDir()
	wadPath := filepath.Join(tmpDir, "Root.wad")
	if err := os.WriteFile(wadPath, wadBytes, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	wad, err := OpenWad(wadPath)
	if err != nil {
		t.Fatalf("OpenWad: %v", err)
	}
	for _, entry := range entries {
		rc, err := wad.OpenFile(entry.name)
		if err != nil {
			t.Fatalf("OpenFile(%s): %v", entry.name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("ReadAll(%s): %v", entry.name, err)
		}
		if string(data) != string(entry.data) {
			t.Fatalf("unexpected data for %s: got %q want %q", entry.name, data, entry.data)
		}
	}
	if _, err := wad.OpenFile("missing"); !errors.Is(err, ErrFileNotFound) {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestManagerLifecycle(t *testing.T) {
	entries := []wadEntry{{name: "foo.bin", data: []byte("abc"), compressed: true}}
	wadBytes := buildTestWad(t, entries)
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "Data", "GameData")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	wadPath := filepath.Join(dataDir, "Root.wad")
	if err := os.WriteFile(wadPath, wadBytes, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	manager := NewManager(tmpDir)
	data, err := manager.GetFileBytes("foo.bin")
	if err != nil {
		t.Fatalf("GetFileBytes: %v", err)
	}
	if string(data) != "abc" {
		t.Fatalf("unexpected data %q", data)
	}

	customWadPath := filepath.Join(tmpDir, "custom.wad")
	if err := os.WriteFile(customWadPath, wadBytes, 0o644); err != nil {
		t.Fatalf("WriteFile custom: %v", err)
	}
	if _, err := manager.AddCustomWad(customWadPath); err != nil {
		t.Fatalf("AddCustomWad: %v", err)
	}

	if _, err := manager.OpenFile("|custom|foo.bin"); err != nil {
		t.Fatalf("OpenFile custom: %v", err)
	}
}
