package io

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// ErrUnexpectedEOF is returned when the buffer does not have enough data to fulfil a read request.
var ErrUnexpectedEOF = errors.New("bytebuffer: unexpected end of buffer")

// ByteBuffer provides little-endian binary encoding helpers with bit-level packing support.
type ByteBuffer struct {
	data       []byte
	rpos       int
	rbitPos    uint8
	rbitBuffer uint8
	wbitPos    uint8
	wbitBuffer uint8
}

// NewByteBuffer constructs an empty buffer ready for writing.
func NewByteBuffer() *ByteBuffer {
	return &ByteBuffer{wbitPos: 8}
}

// NewReader constructs a buffer for reading from the provided data slice. The slice is copied to avoid
// callers mutating the underlying data while reads are in progress.
func NewReader(data []byte) *ByteBuffer {
	dup := make([]byte, len(data))
	copy(dup, data)
	return &ByteBuffer{data: dup, wbitPos: 8, rbitPos: 8}
}

// Bytes returns a copy of the written data.
func (b *ByteBuffer) Bytes() []byte {
	b.flushBits()
	out := make([]byte, len(b.data))
	copy(out, b.data)
	return out
}

// WriteInt8 writes a signed byte to the buffer.
func (b *ByteBuffer) WriteInt8(v int8) {
	b.flushBits()
	b.data = append(b.data, byte(v))
}

// WriteUint8 writes an unsigned byte to the buffer.
func (b *ByteBuffer) WriteUint8(v uint8) {
	b.flushBits()
	b.data = append(b.data, v)
}

// WriteInt16 writes a little-endian int16 to the buffer.
func (b *ByteBuffer) WriteInt16(v int16) {
	b.flushBits()
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], uint16(v))
	b.data = append(b.data, tmp[:]...)
}

// WriteUint16 writes a little-endian uint16 to the buffer.
func (b *ByteBuffer) WriteUint16(v uint16) {
	b.flushBits()
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], v)
	b.data = append(b.data, tmp[:]...)
}

// WriteInt32 writes a little-endian int32 to the buffer.
func (b *ByteBuffer) WriteInt32(v int32) {
	b.flushBits()
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], uint32(v))
	b.data = append(b.data, tmp[:]...)
}

// WriteUint32 writes a little-endian uint32 to the buffer.
func (b *ByteBuffer) WriteUint32(v uint32) {
	b.flushBits()
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	b.data = append(b.data, tmp[:]...)
}

// WriteInt64 writes a little-endian int64 to the buffer.
func (b *ByteBuffer) WriteInt64(v int64) {
	b.flushBits()
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], uint64(v))
	b.data = append(b.data, tmp[:]...)
}

// WriteUint64 writes a little-endian uint64 to the buffer.
func (b *ByteBuffer) WriteUint64(v uint64) {
	b.flushBits()
	var tmp [8]byte
	binary.LittleEndian.PutUint64(tmp[:], v)
	b.data = append(b.data, tmp[:]...)
}

// WriteFloat32 writes an IEEE 754 float32 in little-endian order.
func (b *ByteBuffer) WriteFloat32(v float32) {
	b.WriteUint32(math.Float32bits(v))
}

// WriteFloat64 writes an IEEE 754 float64 in little-endian order.
func (b *ByteBuffer) WriteFloat64(v float64) {
	b.WriteUint64(math.Float64bits(v))
}

// WriteBool writes a single byte representing the boolean value.
func (b *ByteBuffer) WriteBool(v bool) {
	if v {
		b.WriteUint8(1)
		return
	}
	b.WriteUint8(0)
}

// WriteBytes writes the provided bytes into the buffer.
func (b *ByteBuffer) WriteBytes(v []byte) {
	if len(v) == 0 {
		return
	}
	b.flushBits()
	b.data = append(b.data, v...)
}

// WriteCString writes a null-terminated UTF-8 string to the buffer.
func (b *ByteBuffer) WriteCString(v string) {
	if len(v) > 0 {
		b.WriteBytes([]byte(v))
	}
	b.WriteUint8(0)
}

// WriteString writes a UTF-8 string without a terminator.
func (b *ByteBuffer) WriteString(v string) {
	if len(v) == 0 {
		return
	}
	b.WriteBytes([]byte(v))
}

// WriteBit writes a single bit into the buffer using the bit packing rules from the original Open101 buffer.
func (b *ByteBuffer) WriteBit(bit bool) {
	if bit {
		b.writeBitRaw(1)
	} else {
		b.writeBitRaw(0)
	}
}

// WriteBits writes the lowest count bits of value into the buffer.
func (b *ByteBuffer) WriteBits(value uint64, count int) {
	if count < 0 || count > 64 {
		panic("bytebuffer: invalid bit count")
	}
	for i := 0; i < count; i++ {
		b.writeBitRaw(uint8((value >> i) & 1))
	}
}

func (b *ByteBuffer) writeBitRaw(bit uint8) {
	if bit != 0 && bit != 1 {
		panic("bytebuffer: bit must be 0 or 1")
	}
	if b.wbitPos == 0 {
		b.flushBits()
	}
	b.wbitPos--
	if bit == 1 {
		b.wbitBuffer |= 1 << b.wbitPos
	}
	if b.wbitPos == 0 {
		b.flushBits()
	}
}

func (b *ByteBuffer) flushBits() {
	if b.wbitPos == 8 {
		return
	}
	b.data = append(b.data, reverseByte(b.wbitBuffer))
	b.wbitBuffer = 0
	b.wbitPos = 8
}

// ResetReader positions the reader at the beginning of the buffer.
func (b *ByteBuffer) ResetReader() {
	b.rpos = 0
	b.rbitPos = 8
	b.rbitBuffer = 0
}

// Remaining returns the number of unread bytes (excluding buffered bits).
func (b *ByteBuffer) Remaining() int {
	if b.rpos >= len(b.data) {
		return 0
	}
	return len(b.data) - b.rpos
}

func (b *ByteBuffer) discardReadBits() {
	b.rbitPos = 8
	b.rbitBuffer = 0
}

// ReadInt8 reads a signed byte from the buffer.
func (b *ByteBuffer) ReadInt8() (int8, error) {
	v, err := b.ReadUint8()
	return int8(v), err
}

// ReadUint8 reads an unsigned byte from the buffer.
func (b *ByteBuffer) ReadUint8() (uint8, error) {
	b.discardReadBits()
	if b.rpos >= len(b.data) {
		return 0, ErrUnexpectedEOF
	}
	v := b.data[b.rpos]
	b.rpos++
	return v, nil
}

// ReadInt16 reads a little-endian int16 from the buffer.
func (b *ByteBuffer) ReadInt16() (int16, error) {
	v, err := b.ReadUint16()
	return int16(v), err
}

// ReadUint16 reads a little-endian uint16 from the buffer.
func (b *ByteBuffer) ReadUint16() (uint16, error) {
	b.discardReadBits()
	if b.rpos+2 > len(b.data) {
		return 0, ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint16(b.data[b.rpos:])
	b.rpos += 2
	return v, nil
}

// ReadInt32 reads a little-endian int32 from the buffer.
func (b *ByteBuffer) ReadInt32() (int32, error) {
	v, err := b.ReadUint32()
	return int32(v), err
}

// ReadUint32 reads a little-endian uint32 from the buffer.
func (b *ByteBuffer) ReadUint32() (uint32, error) {
	b.discardReadBits()
	if b.rpos+4 > len(b.data) {
		return 0, ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint32(b.data[b.rpos:])
	b.rpos += 4
	return v, nil
}

// ReadInt64 reads a little-endian int64 from the buffer.
func (b *ByteBuffer) ReadInt64() (int64, error) {
	v, err := b.ReadUint64()
	return int64(v), err
}

// ReadUint64 reads a little-endian uint64 from the buffer.
func (b *ByteBuffer) ReadUint64() (uint64, error) {
	b.discardReadBits()
	if b.rpos+8 > len(b.data) {
		return 0, ErrUnexpectedEOF
	}
	v := binary.LittleEndian.Uint64(b.data[b.rpos:])
	b.rpos += 8
	return v, nil
}

// ReadFloat32 reads an IEEE 754 float32.
func (b *ByteBuffer) ReadFloat32() (float32, error) {
	v, err := b.ReadUint32()
	return math.Float32frombits(v), err
}

// ReadFloat64 reads an IEEE 754 float64.
func (b *ByteBuffer) ReadFloat64() (float64, error) {
	v, err := b.ReadUint64()
	return math.Float64frombits(v), err
}

// ReadBool reads a boolean.
func (b *ByteBuffer) ReadBool() (bool, error) {
	v, err := b.ReadUint8()
	if err != nil {
		return false, err
	}
	return v != 0, nil
}

// ReadBytes returns a copy of count bytes.
func (b *ByteBuffer) ReadBytes(count int) ([]byte, error) {
	b.discardReadBits()
	if count < 0 {
		return nil, errors.New("bytebuffer: negative count")
	}
	if b.rpos+count > len(b.data) {
		return nil, ErrUnexpectedEOF
	}
	out := make([]byte, count)
	copy(out, b.data[b.rpos:b.rpos+count])
	b.rpos += count
	return out, nil
}

// ReadCString reads a null-terminated UTF-8 string.
func (b *ByteBuffer) ReadCString() (string, error) {
	b.discardReadBits()
	start := b.rpos
	for {
		if b.rpos >= len(b.data) {
			return "", ErrUnexpectedEOF
		}
		if b.data[b.rpos] == 0 {
			s := string(b.data[start:b.rpos])
			b.rpos++
			return s, nil
		}
		b.rpos++
	}
}

// ReadString reads exactly length bytes as a UTF-8 string.
func (b *ByteBuffer) ReadString(length int) (string, error) {
	bytes, err := b.ReadBytes(length)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ReadBit reads a single bit using the Open101 packing rules.
func (b *ByteBuffer) ReadBit() (bool, error) {
	if b.rbitPos >= 8 {
		if b.rpos >= len(b.data) {
			return false, ErrUnexpectedEOF
		}
		b.rbitBuffer = reverseByte(b.data[b.rpos])
		b.rpos++
		b.rbitPos = 0
	}
	bit := (b.rbitBuffer >> (7 - b.rbitPos)) & 1
	b.rbitPos++
	return bit == 1, nil
}

// ReadBits reads count bits and packs them into the lowest bits of a uint64.
func (b *ByteBuffer) ReadBits(count int) (uint64, error) {
	if count < 0 || count > 64 {
		return 0, errors.New("bytebuffer: invalid bit count")
	}
	var value uint64
	for i := 0; i < count; i++ {
		bit, err := b.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit {
			value |= 1 << i
		}
	}
	return value, nil
}

// reverseByte mirrors the bits inside b.
func reverseByte(b byte) byte {
	b = (b&0xF0)>>4 | (b&0x0F)<<4
	b = (b&0xCC)>>2 | (b&0x33)<<2
	b = (b&0xAA)>>1 | (b&0x55)<<1
	return b
}

// Drain copies all remaining unread bytes to the provided writer.
func (b *ByteBuffer) Drain(w io.Writer) (int64, error) {
	b.discardReadBits()
	if b.rpos >= len(b.data) {
		return 0, nil
	}
	n, err := w.Write(b.data[b.rpos:])
	b.rpos = len(b.data)
	return int64(n), err
}
