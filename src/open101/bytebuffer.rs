use std::io::Write;

use crate::{Error, Result};

#[derive(Clone, Debug)]
pub struct ByteBuffer {
    data: Vec<u8>,
    read_pos: usize,
    read_bit_pos: u8,
    read_bit_buffer: u8,
    write_bit_pos: u8,
    write_bit_buffer: u8,
}

impl Default for ByteBuffer {
    fn default() -> Self {
        Self::new()
    }
}

impl ByteBuffer {
    pub fn new() -> Self {
        Self {
            data: Vec::new(),
            read_pos: 0,
            read_bit_pos: 8,
            read_bit_buffer: 0,
            write_bit_pos: 8,
            write_bit_buffer: 0,
        }
    }

    pub fn from_bytes(data: &[u8]) -> Self {
        Self {
            data: data.to_vec(),
            read_pos: 0,
            read_bit_pos: 8,
            read_bit_buffer: 0,
            write_bit_pos: 8,
            write_bit_buffer: 0,
        }
    }

    pub fn bytes(&mut self) -> Vec<u8> {
        self.flush_bits();
        self.data.clone()
    }

    pub fn write_i8(&mut self, value: i8) {
        self.write_u8(value as u8);
    }

    pub fn write_u8(&mut self, value: u8) {
        self.flush_bits();
        self.data.push(value);
    }

    pub fn write_i16(&mut self, value: i16) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_u16(&mut self, value: u16) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_i32(&mut self, value: i32) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_u32(&mut self, value: u32) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_i64(&mut self, value: i64) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_u64(&mut self, value: u64) {
        self.write_bytes(&value.to_le_bytes());
    }

    pub fn write_f32(&mut self, value: f32) {
        self.write_u32(value.to_bits());
    }

    pub fn write_f64(&mut self, value: f64) {
        self.write_u64(value.to_bits());
    }

    pub fn write_bool(&mut self, value: bool) {
        self.write_u8(u8::from(value));
    }

    pub fn write_bytes(&mut self, value: &[u8]) {
        if value.is_empty() {
            return;
        }
        self.flush_bits();
        self.data.extend_from_slice(value);
    }

    pub fn write_c_string(&mut self, value: &str) {
        self.write_bytes(value.as_bytes());
        self.write_u8(0);
    }

    pub fn write_string(&mut self, value: &str) {
        self.write_bytes(value.as_bytes());
    }

    pub fn write_bit(&mut self, value: bool) {
        self.write_bit_raw(u8::from(value));
    }

    pub fn write_bits(&mut self, value: u64, count: usize) -> Result<()> {
        if count > 64 {
            return Err(Error::message("bytebuffer: invalid bit count"));
        }
        for index in 0..count {
            self.write_bit_raw(((value >> index) & 1) as u8);
        }
        Ok(())
    }

    pub fn reset_reader(&mut self) {
        self.read_pos = 0;
        self.read_bit_pos = 8;
        self.read_bit_buffer = 0;
    }

    pub fn remaining(&self) -> usize {
        self.data.len().saturating_sub(self.read_pos)
    }

    pub fn read_i8(&mut self) -> Result<i8> {
        Ok(self.read_u8()? as i8)
    }

    pub fn read_u8(&mut self) -> Result<u8> {
        self.discard_read_bits();
        let value = *self.data.get(self.read_pos).ok_or_else(unexpected_eof)?;
        self.read_pos += 1;
        Ok(value)
    }

    pub fn read_i16(&mut self) -> Result<i16> {
        Ok(i16::from_le_bytes(self.read_array()?))
    }

    pub fn read_u16(&mut self) -> Result<u16> {
        Ok(u16::from_le_bytes(self.read_array()?))
    }

    pub fn read_i32(&mut self) -> Result<i32> {
        Ok(i32::from_le_bytes(self.read_array()?))
    }

    pub fn read_u32(&mut self) -> Result<u32> {
        Ok(u32::from_le_bytes(self.read_array()?))
    }

    pub fn read_i64(&mut self) -> Result<i64> {
        Ok(i64::from_le_bytes(self.read_array()?))
    }

    pub fn read_u64(&mut self) -> Result<u64> {
        Ok(u64::from_le_bytes(self.read_array()?))
    }

    pub fn read_f32(&mut self) -> Result<f32> {
        Ok(f32::from_bits(self.read_u32()?))
    }

    pub fn read_f64(&mut self) -> Result<f64> {
        Ok(f64::from_bits(self.read_u64()?))
    }

    pub fn read_bool(&mut self) -> Result<bool> {
        Ok(self.read_u8()? != 0)
    }

    pub fn read_bytes(&mut self, count: usize) -> Result<Vec<u8>> {
        self.discard_read_bits();
        let end = self
            .read_pos
            .checked_add(count)
            .filter(|end| *end <= self.data.len())
            .ok_or_else(unexpected_eof)?;
        let value = self.data[self.read_pos..end].to_vec();
        self.read_pos = end;
        Ok(value)
    }

    pub fn read_c_string(&mut self) -> Result<String> {
        self.discard_read_bits();
        let start = self.read_pos;
        let relative_end = self.data[start..]
            .iter()
            .position(|byte| *byte == 0)
            .ok_or_else(unexpected_eof)?;
        let end = start + relative_end;
        self.read_pos = end + 1;
        Ok(String::from_utf8_lossy(&self.data[start..end]).into_owned())
    }

    pub fn read_string(&mut self, length: usize) -> Result<String> {
        Ok(String::from_utf8_lossy(&self.read_bytes(length)?).into_owned())
    }

    pub fn read_bit(&mut self) -> Result<bool> {
        if self.read_bit_pos >= 8 {
            let byte = *self.data.get(self.read_pos).ok_or_else(unexpected_eof)?;
            self.read_pos += 1;
            self.read_bit_buffer = reverse_byte(byte);
            self.read_bit_pos = 0;
        }

        let bit = (self.read_bit_buffer >> (7 - self.read_bit_pos)) & 1;
        self.read_bit_pos += 1;
        Ok(bit == 1)
    }

    pub fn read_bits(&mut self, count: usize) -> Result<u64> {
        if count > 64 {
            return Err(Error::message("bytebuffer: invalid bit count"));
        }

        let mut value = 0_u64;
        for index in 0..count {
            if self.read_bit()? {
                value |= 1_u64 << index;
            }
        }
        Ok(value)
    }

    pub fn drain<W: Write>(&mut self, writer: &mut W) -> Result<u64> {
        self.discard_read_bits();
        if self.read_pos >= self.data.len() {
            return Ok(0);
        }

        let remaining = &self.data[self.read_pos..];
        writer.write_all(remaining)?;
        let written = remaining.len() as u64;
        self.read_pos = self.data.len();
        Ok(written)
    }

    fn read_array<const N: usize>(&mut self) -> Result<[u8; N]> {
        self.discard_read_bits();
        let end = self
            .read_pos
            .checked_add(N)
            .filter(|end| *end <= self.data.len())
            .ok_or_else(unexpected_eof)?;
        let mut output = [0_u8; N];
        output.copy_from_slice(&self.data[self.read_pos..end]);
        self.read_pos = end;
        Ok(output)
    }

    fn write_bit_raw(&mut self, bit: u8) {
        debug_assert!(bit <= 1);
        if self.write_bit_pos == 0 {
            self.flush_bits();
        }
        self.write_bit_pos -= 1;
        if bit == 1 {
            self.write_bit_buffer |= 1 << self.write_bit_pos;
        }
        if self.write_bit_pos == 0 {
            self.flush_bits();
        }
    }

    fn flush_bits(&mut self) {
        if self.write_bit_pos == 8 {
            return;
        }
        self.data.push(reverse_byte(self.write_bit_buffer));
        self.write_bit_buffer = 0;
        self.write_bit_pos = 8;
    }

    fn discard_read_bits(&mut self) {
        self.read_bit_pos = 8;
        self.read_bit_buffer = 0;
    }
}

fn unexpected_eof() -> Error {
    Error::message("bytebuffer: unexpected end of buffer")
}

fn reverse_byte(mut byte: u8) -> u8 {
    byte = (byte & 0xf0) >> 4 | (byte & 0x0f) << 4;
    byte = (byte & 0xcc) >> 2 | (byte & 0x33) << 2;
    (byte & 0xaa) >> 1 | (byte & 0x55) << 1
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn scalar_round_trip() {
        let mut buffer = ByteBuffer::new();
        buffer.write_i8(-7);
        buffer.write_u8(250);
        buffer.write_i16(-12345);
        buffer.write_u16(50000);
        buffer.write_i32(-123456789);
        buffer.write_u32(3456789012);
        buffer.write_i64(-567890123456);
        buffer.write_u64(987654321012345678);
        buffer.write_f32(f32::MIN_POSITIVE);
        buffer.write_f64(std::f64::consts::PI);
        buffer.write_bool(true);
        buffer.write_bool(false);
        buffer.write_c_string("hello");
        buffer.write_string("world");

        let raw = buffer.bytes();
        let mut reader = ByteBuffer::from_bytes(&raw);
        assert_eq!(reader.read_i8().unwrap(), -7);
        assert_eq!(reader.read_u8().unwrap(), 250);
        assert_eq!(reader.read_i16().unwrap(), -12345);
        assert_eq!(reader.read_u16().unwrap(), 50000);
        assert_eq!(reader.read_i32().unwrap(), -123456789);
        assert_eq!(reader.read_u32().unwrap(), 3456789012);
        assert_eq!(reader.read_i64().unwrap(), -567890123456);
        assert_eq!(reader.read_u64().unwrap(), 987654321012345678);
        assert_eq!(reader.read_f32().unwrap(), f32::MIN_POSITIVE);
        assert_eq!(reader.read_f64().unwrap(), std::f64::consts::PI);
        assert!(reader.read_bool().unwrap());
        assert!(!reader.read_bool().unwrap());
        assert_eq!(reader.read_c_string().unwrap(), "hello");
        assert_eq!(reader.read_string(5).unwrap(), "world");
        assert_eq!(reader.remaining(), 0);
    }

    #[test]
    fn bit_packing_round_trip() {
        let mut buffer = ByteBuffer::new();
        buffer.write_bits(0b101101, 6).unwrap();
        buffer.write_bit(true);
        buffer.write_bit(false);
        buffer.write_u8(0xaa);
        let raw = buffer.bytes();

        let mut reader = ByteBuffer::from_bytes(&raw);
        assert_eq!(reader.read_bits(6).unwrap(), 0b101101);
        assert!(reader.read_bit().unwrap());
        assert!(!reader.read_bit().unwrap());
        assert_eq!(reader.read_u8().unwrap(), 0xaa);
    }

    #[test]
    fn drain_writes_remaining_bytes() {
        let mut buffer = ByteBuffer::new();
        buffer.write_u32(0xaabbccdd);
        let raw = buffer.bytes();
        let mut reader = ByteBuffer::from_bytes(&raw);
        reader.read_u16().unwrap();

        let mut output = Vec::new();
        reader.drain(&mut output).unwrap();
        assert_eq!(output, raw[2..]);
    }

    #[test]
    fn reports_unexpected_eof() {
        let mut reader = ByteBuffer::from_bytes(&[1]);
        assert!(reader.read_u16().is_err());
    }
}
