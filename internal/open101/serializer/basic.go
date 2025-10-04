package serializer

import (
	bufferio "github.com/cybellereaper/open101/internal/open101/io"
)

// The BasicBinarySerializer mirrors the helper surface in the reference implementation while leveraging
// the strongly-typed Go byte buffer.

func ReadBit(buf *bufferio.ByteBuffer) (bool, error)             { return buf.ReadBit() }
func ReadInt8(buf *bufferio.ByteBuffer) (int8, error)            { return buf.ReadInt8() }
func ReadUint8(buf *bufferio.ByteBuffer) (uint8, error)          { return buf.ReadUint8() }
func ReadInt16(buf *bufferio.ByteBuffer) (int16, error)          { return buf.ReadInt16() }
func ReadUint16(buf *bufferio.ByteBuffer) (uint16, error)        { return buf.ReadUint16() }
func ReadInt32(buf *bufferio.ByteBuffer) (int32, error)          { return buf.ReadInt32() }
func ReadUint32(buf *bufferio.ByteBuffer) (uint32, error)        { return buf.ReadUint32() }
func ReadInt64(buf *bufferio.ByteBuffer) (int64, error)          { return buf.ReadInt64() }
func ReadUint64(buf *bufferio.ByteBuffer) (uint64, error)        { return buf.ReadUint64() }
func ReadFloat32(buf *bufferio.ByteBuffer) (float32, error)      { return buf.ReadFloat32() }
func ReadFloat64(buf *bufferio.ByteBuffer) (float64, error)      { return buf.ReadFloat64() }
func ReadBool(buf *bufferio.ByteBuffer) (bool, error)            { return buf.ReadBool() }
func ReadCString(buf *bufferio.ByteBuffer) (string, error)       { return buf.ReadCString() }
func ReadString(buf *bufferio.ByteBuffer, n int) (string, error) { return buf.ReadString(n) }
func ReadBits(buf *bufferio.ByteBuffer, count int) (uint64, error) {
	return buf.ReadBits(count)
}

func WriteBool(buf *bufferio.ByteBuffer, v bool)          { buf.WriteBool(v) }
func WriteInt8(buf *bufferio.ByteBuffer, v int8)          { buf.WriteInt8(v) }
func WriteUint8(buf *bufferio.ByteBuffer, v uint8)        { buf.WriteUint8(v) }
func WriteInt16(buf *bufferio.ByteBuffer, v int16)        { buf.WriteInt16(v) }
func WriteUint16(buf *bufferio.ByteBuffer, v uint16)      { buf.WriteUint16(v) }
func WriteInt32(buf *bufferio.ByteBuffer, v int32)        { buf.WriteInt32(v) }
func WriteUint32(buf *bufferio.ByteBuffer, v uint32)      { buf.WriteUint32(v) }
func WriteInt64(buf *bufferio.ByteBuffer, v int64)        { buf.WriteInt64(v) }
func WriteUint64(buf *bufferio.ByteBuffer, v uint64)      { buf.WriteUint64(v) }
func WriteFloat32(buf *bufferio.ByteBuffer, v float32)    { buf.WriteFloat32(v) }
func WriteFloat64(buf *bufferio.ByteBuffer, v float64)    { buf.WriteFloat64(v) }
func WriteCString(buf *bufferio.ByteBuffer, v string)     { buf.WriteCString(v) }
func WriteString(buf *bufferio.ByteBuffer, v string)      { buf.WriteString(v) }
func WriteBits(buf *bufferio.ByteBuffer, v uint64, c int) { buf.WriteBits(v, c) }

// ReadGID reads a global identifier from the buffer.
func ReadGID(buf *bufferio.ByteBuffer) (GID, error) {
	raw, err := buf.ReadUint64()
	if err != nil {
		return GID{}, err
	}
	return GID{Value: raw}, nil
}

// WriteGID writes a global identifier to the buffer.
func WriteGID(buf *bufferio.ByteBuffer, gid GID) {
	buf.WriteUint64(gid.Value)
}
