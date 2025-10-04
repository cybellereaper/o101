package io

import (
	"bytes"
	"errors"
	"math"
	"testing"
)

func TestByteBufferReadWriteScalars(t *testing.T) {
	buf := NewByteBuffer()
	buf.WriteInt8(-7)
	buf.WriteUint8(250)
	buf.WriteInt16(-12345)
	buf.WriteUint16(50000)
	buf.WriteInt32(-123456789)
	buf.WriteUint32(3456789012)
	buf.WriteInt64(-567890123456)
	buf.WriteUint64(987654321012345678)
	buf.WriteFloat32(math.SmallestNonzeroFloat32)
	buf.WriteFloat64(math.Pi)
	buf.WriteBool(true)
	buf.WriteBool(false)
	buf.WriteCString("hello")
	buf.WriteString("world")
	raw := buf.Bytes()

	reader := NewReader(raw)
	if v, err := reader.ReadInt8(); err != nil || v != -7 {
		t.Fatalf("ReadInt8 = %v, %v", v, err)
	}
	if v, err := reader.ReadUint8(); err != nil || v != 250 {
		t.Fatalf("ReadUint8 = %v, %v", v, err)
	}
	if v, err := reader.ReadInt16(); err != nil || v != -12345 {
		t.Fatalf("ReadInt16 = %v, %v", v, err)
	}
	if v, err := reader.ReadUint16(); err != nil || v != 50000 {
		t.Fatalf("ReadUint16 = %v, %v", v, err)
	}
	if v, err := reader.ReadInt32(); err != nil || v != -123456789 {
		t.Fatalf("ReadInt32 = %v, %v", v, err)
	}
	if v, err := reader.ReadUint32(); err != nil || v != 3456789012 {
		t.Fatalf("ReadUint32 = %v, %v", v, err)
	}
	if v, err := reader.ReadInt64(); err != nil || v != -567890123456 {
		t.Fatalf("ReadInt64 = %v, %v", v, err)
	}
	if v, err := reader.ReadUint64(); err != nil || v != 987654321012345678 {
		t.Fatalf("ReadUint64 = %v, %v", v, err)
	}
	if v, err := reader.ReadFloat32(); err != nil || v != math.SmallestNonzeroFloat32 {
		t.Fatalf("ReadFloat32 = %v, %v", v, err)
	}
	if v, err := reader.ReadFloat64(); err != nil || v != math.Pi {
		t.Fatalf("ReadFloat64 = %v, %v", v, err)
	}
	if v, err := reader.ReadBool(); err != nil || v != true {
		t.Fatalf("ReadBool = %v, %v", v, err)
	}
	if v, err := reader.ReadBool(); err != nil || v != false {
		t.Fatalf("ReadBool = %v, %v", v, err)
	}
	if v, err := reader.ReadCString(); err != nil || v != "hello" {
		t.Fatalf("ReadCString = %v, %v", v, err)
	}
	if v, err := reader.ReadString(5); err != nil || v != "world" {
		t.Fatalf("ReadString = %v, %v", v, err)
	}
	if reader.Remaining() != 0 {
		t.Fatalf("expected buffer to be fully consumed, remaining=%d", reader.Remaining())
	}
}

func TestByteBufferBitPacking(t *testing.T) {
	buf := NewByteBuffer()
	buf.WriteBits(0b101101, 6)
	buf.WriteBit(true)
	buf.WriteBit(false)
	buf.WriteUint8(0xAA)
	raw := buf.Bytes()

	reader := NewReader(raw)
	bits, err := reader.ReadBits(6)
	if err != nil {
		t.Fatalf("ReadBits error: %v", err)
	}
	if bits != 0b101101 {
		t.Fatalf("ReadBits = %b", bits)
	}
	b, err := reader.ReadBit()
	if err != nil || !b {
		t.Fatalf("ReadBit expected true: %v, %v", b, err)
	}
	b, err = reader.ReadBit()
	if err != nil || b {
		t.Fatalf("ReadBit expected false: %v, %v", b, err)
	}
	v, err := reader.ReadUint8()
	if err != nil || v != 0xAA {
		t.Fatalf("ReadUint8 = %02X, %v", v, err)
	}
}

func TestByteBufferDrain(t *testing.T) {
	buf := NewByteBuffer()
	buf.WriteUint32(0xAABBCCDD)
	raw := buf.Bytes()

	reader := NewReader(raw)
	if _, err := reader.ReadUint16(); err != nil {
		t.Fatalf("ReadUint16 error: %v", err)
	}
	var out bytes.Buffer
	if _, err := reader.Drain(&out); err != nil {
		t.Fatalf("Drain error: %v", err)
	}
	if got := out.Bytes(); !bytes.Equal(got, raw[2:]) {
		t.Fatalf("Drain = %v, want %v", got, raw[2:])
	}
}

func TestByteBufferUnexpectedEOF(t *testing.T) {
	reader := NewReader([]byte{0x01})
	if _, err := reader.ReadUint16(); !errors.Is(err, ErrUnexpectedEOF) {
		t.Fatalf("expected unexpected EOF error, got %v", err)
	}
}
