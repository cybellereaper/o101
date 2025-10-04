package serializer

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// ByteString wraps a UTF-8 byte slice and mirrors the ergonomics of the original implicit conversions.
type ByteString struct {
	data []byte
}

// NewByteString builds a ByteString from the supplied slice, copying the data to keep the wrapper immutable.
func NewByteString(b []byte) ByteString {
	if len(b) == 0 {
		return ByteString{}
	}
	dup := make([]byte, len(b))
	copy(dup, b)
	return ByteString{data: dup}
}

// Bytes returns a copy of the underlying bytes.
func (b ByteString) Bytes() []byte {
	if len(b.data) == 0 {
		return nil
	}
	dup := make([]byte, len(b.data))
	copy(dup, b.data)
	return dup
}

// String decodes the bytes as UTF-8 text.
func (b ByteString) String() string {
	return string(b.data)
}

// Hex renders the bytes as a hexadecimal string, mirroring useful debugging helpers from the C# codebase.
func (b ByteString) Hex() string {
	return hex.EncodeToString(b.data)
}

// IsZero reports whether the wrapper contains no data.
func (b ByteString) IsZero() bool {
	return len(b.data) == 0
}

// MarshalText implements encoding.TextMarshaler.
func (b ByteString) MarshalText() ([]byte, error) {
	return b.Bytes(), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (b *ByteString) UnmarshalText(text []byte) error {
	*b = NewByteString(text)
	return nil
}

// GID represents the globally unique identifier type used across KingsIsle packets.
type GID struct {
	Value uint64
}

// String returns the canonical hexadecimal representation.
func (g GID) String() string {
	return fmt.Sprintf("0x%016X", g.Value)
}

// IsZero reports whether the identifier is unset.
func (g GID) IsZero() bool {
	return g.Value == 0
}

// MarshalText implements encoding.TextMarshaler.
func (g GID) MarshalText() ([]byte, error) {
	return []byte(g.String()), nil
}

// UnmarshalText parses the canonical hexadecimal format.
func (g *GID) UnmarshalText(text []byte) error {
	cleaned := strings.TrimSpace(string(text))
	cleaned = strings.TrimPrefix(cleaned, "0x")
	cleaned = strings.TrimPrefix(cleaned, "0X")
	if cleaned == "" {
		g.Value = 0
		return nil
	}
	value, err := strconv.ParseUint(cleaned, 16, 64)
	if err != nil {
		return err
	}
	g.Value = value
	return nil
}

// ParseGID converts a hexadecimal string to a GID.
func ParseGID(input string) (GID, error) {
	var g GID
	if err := g.UnmarshalText([]byte(input)); err != nil {
		return GID{}, err
	}
	return g, nil
}
