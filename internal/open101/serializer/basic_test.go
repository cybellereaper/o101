package serializer

import (
	"testing"

	bufferio "github.com/cybellereaper/wizturtle/v2/internal/open101/io"
)

func TestBasicBinarySerializerRoundTrip(t *testing.T) {
	buf := bufferio.NewByteBuffer()
	WriteInt8(buf, -8)
	WriteUint8(buf, 200)
	WriteInt16(buf, -1234)
	WriteUint16(buf, 54321)
	WriteInt32(buf, -555666777)
	WriteUint32(buf, 3332221110)
	WriteInt64(buf, -987654321012345)
	WriteUint64(buf, 111222333444555666)
	WriteFloat32(buf, 123.5)
	WriteFloat64(buf, 98765.4321)
	WriteBool(buf, true)
	WriteBool(buf, false)
	WriteCString(buf, "hello")
	WriteString(buf, "world")
	WriteBits(buf, 0b1101, 4)
	WriteGID(buf, GID{Value: 0x1234})

	raw := buf.Bytes()
	reader := bufferio.NewReader(raw)

	if v, err := ReadInt8(reader); err != nil || v != -8 {
		t.Fatalf("ReadInt8 = %v, %v", v, err)
	}
	if v, err := ReadUint8(reader); err != nil || v != 200 {
		t.Fatalf("ReadUint8 = %v, %v", v, err)
	}
	if v, err := ReadInt16(reader); err != nil || v != -1234 {
		t.Fatalf("ReadInt16 = %v, %v", v, err)
	}
	if v, err := ReadUint16(reader); err != nil || v != 54321 {
		t.Fatalf("ReadUint16 = %v, %v", v, err)
	}
	if v, err := ReadInt32(reader); err != nil || v != -555666777 {
		t.Fatalf("ReadInt32 = %v, %v", v, err)
	}
	if v, err := ReadUint32(reader); err != nil || v != 3332221110 {
		t.Fatalf("ReadUint32 = %v, %v", v, err)
	}
	if v, err := ReadInt64(reader); err != nil || v != -987654321012345 {
		t.Fatalf("ReadInt64 = %v, %v", v, err)
	}
	if v, err := ReadUint64(reader); err != nil || v != 111222333444555666 {
		t.Fatalf("ReadUint64 = %v, %v", v, err)
	}
	if v, err := ReadFloat32(reader); err != nil || v != 123.5 {
		t.Fatalf("ReadFloat32 = %v, %v", v, err)
	}
	if v, err := ReadFloat64(reader); err != nil || v != 98765.4321 {
		t.Fatalf("ReadFloat64 = %v, %v", v, err)
	}
	if v, err := ReadBool(reader); err != nil || v != true {
		t.Fatalf("ReadBool = %v, %v", v, err)
	}
	if v, err := ReadBool(reader); err != nil || v != false {
		t.Fatalf("ReadBool = %v, %v", v, err)
	}
	if s, err := ReadCString(reader); err != nil || s != "hello" {
		t.Fatalf("ReadCString = %q, %v", s, err)
	}
	if s, err := ReadString(reader, 5); err != nil || s != "world" {
		t.Fatalf("ReadString = %q, %v", s, err)
	}
	if bits, err := ReadBits(reader, 4); err != nil || bits != 0b1101 {
		t.Fatalf("ReadBits = %b, %v", bits, err)
	}
	if gid, err := ReadGID(reader); err != nil || gid.Value != 0x1234 {
		t.Fatalf("ReadGID = %v, %v", gid, err)
	}
}

func TestByteStringHelpers(t *testing.T) {
	bs := NewByteString([]byte("hello"))
	if bs.IsZero() {
		t.Fatalf("expected non-zero byte string")
	}
	if bs.String() != "hello" {
		t.Fatalf("String = %q", bs.String())
	}
	if hex := bs.Hex(); hex != "68656c6c6f" {
		t.Fatalf("Hex = %q", hex)
	}
	copyBytes := bs.Bytes()
	copyBytes[0] = 'H'
	if bs.String() != "hello" {
		t.Fatalf("Bytes should return copy")
	}
	var roundTrip ByteString
	if err := roundTrip.UnmarshalText([]byte("world")); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if text, err := roundTrip.MarshalText(); err != nil || string(text) != "world" {
		t.Fatalf("MarshalText = %q, %v", text, err)
	}
}

func TestGIDHelpers(t *testing.T) {
	gid := GID{Value: 0xABCDEF1234567890}
	if gid.IsZero() {
		t.Fatalf("unexpected zero GID")
	}
	if gid.String() != "0xABCDEF1234567890" {
		t.Fatalf("String = %s", gid.String())
	}
	text, err := gid.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText: %v", err)
	}
	parsed, err := ParseGID(string(text))
	if err != nil {
		t.Fatalf("ParseGID: %v", err)
	}
	if parsed.Value != gid.Value {
		t.Fatalf("ParseGID mismatch")
	}
	var zero GID
	if err := zero.UnmarshalText([]byte("0x10")); err != nil {
		t.Fatalf("UnmarshalText: %v", err)
	}
	if zero.Value != 0x10 {
		t.Fatalf("UnmarshalText mismatch")
	}
}
