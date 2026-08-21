use std::{fmt, str::FromStr};

use crate::{Error, Result};

use super::bytebuffer::ByteBuffer;

#[derive(Clone, Debug, Default, Eq, PartialEq)]
pub struct ByteString(Vec<u8>);

impl ByteString {
    pub fn new(bytes: &[u8]) -> Self {
        Self(bytes.to_vec())
    }

    pub fn bytes(&self) -> Vec<u8> {
        self.0.clone()
    }

    pub fn string(&self) -> String {
        String::from_utf8_lossy(&self.0).into_owned()
    }

    pub fn hex(&self) -> String {
        to_hex(&self.0)
    }

    pub fn is_zero(&self) -> bool {
        self.0.is_empty()
    }
}

impl From<&str> for ByteString {
    fn from(value: &str) -> Self {
        Self::new(value.as_bytes())
    }
}

#[derive(Clone, Copy, Debug, Default, Eq, Hash, PartialEq)]
pub struct Gid {
    pub value: u64,
}

impl Gid {
    pub fn is_zero(self) -> bool {
        self.value == 0
    }
}

impl fmt::Display for Gid {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "0x{:016X}", self.value)
    }
}

impl FromStr for Gid {
    type Err = Error;

    fn from_str(value: &str) -> Result<Self> {
        let value = value.trim();
        let value = value
            .strip_prefix("0x")
            .or_else(|| value.strip_prefix("0X"))
            .unwrap_or(value);

        if value.is_empty() {
            return Ok(Self::default());
        }

        let value = u64::from_str_radix(value, 16)
            .map_err(|error| Error::message(format!("invalid GID: {error}")))?;
        Ok(Self { value })
    }
}

pub fn read_bit(buffer: &mut ByteBuffer) -> Result<bool> {
    buffer.read_bit()
}

pub fn read_i8(buffer: &mut ByteBuffer) -> Result<i8> {
    buffer.read_i8()
}

pub fn read_u8(buffer: &mut ByteBuffer) -> Result<u8> {
    buffer.read_u8()
}

pub fn read_i16(buffer: &mut ByteBuffer) -> Result<i16> {
    buffer.read_i16()
}

pub fn read_u16(buffer: &mut ByteBuffer) -> Result<u16> {
    buffer.read_u16()
}

pub fn read_i32(buffer: &mut ByteBuffer) -> Result<i32> {
    buffer.read_i32()
}

pub fn read_u32(buffer: &mut ByteBuffer) -> Result<u32> {
    buffer.read_u32()
}

pub fn read_i64(buffer: &mut ByteBuffer) -> Result<i64> {
    buffer.read_i64()
}

pub fn read_u64(buffer: &mut ByteBuffer) -> Result<u64> {
    buffer.read_u64()
}

pub fn read_f32(buffer: &mut ByteBuffer) -> Result<f32> {
    buffer.read_f32()
}

pub fn read_f64(buffer: &mut ByteBuffer) -> Result<f64> {
    buffer.read_f64()
}

pub fn read_bool(buffer: &mut ByteBuffer) -> Result<bool> {
    buffer.read_bool()
}

pub fn read_c_string(buffer: &mut ByteBuffer) -> Result<String> {
    buffer.read_c_string()
}

pub fn read_string(buffer: &mut ByteBuffer, length: usize) -> Result<String> {
    buffer.read_string(length)
}

pub fn read_bits(buffer: &mut ByteBuffer, count: usize) -> Result<u64> {
    buffer.read_bits(count)
}

pub fn read_gid(buffer: &mut ByteBuffer) -> Result<Gid> {
    Ok(Gid {
        value: buffer.read_u64()?,
    })
}

pub fn write_bool(buffer: &mut ByteBuffer, value: bool) {
    buffer.write_bool(value);
}

pub fn write_i8(buffer: &mut ByteBuffer, value: i8) {
    buffer.write_i8(value);
}

pub fn write_u8(buffer: &mut ByteBuffer, value: u8) {
    buffer.write_u8(value);
}

pub fn write_i16(buffer: &mut ByteBuffer, value: i16) {
    buffer.write_i16(value);
}

pub fn write_u16(buffer: &mut ByteBuffer, value: u16) {
    buffer.write_u16(value);
}

pub fn write_i32(buffer: &mut ByteBuffer, value: i32) {
    buffer.write_i32(value);
}

pub fn write_u32(buffer: &mut ByteBuffer, value: u32) {
    buffer.write_u32(value);
}

pub fn write_i64(buffer: &mut ByteBuffer, value: i64) {
    buffer.write_i64(value);
}

pub fn write_u64(buffer: &mut ByteBuffer, value: u64) {
    buffer.write_u64(value);
}

pub fn write_f32(buffer: &mut ByteBuffer, value: f32) {
    buffer.write_f32(value);
}

pub fn write_f64(buffer: &mut ByteBuffer, value: f64) {
    buffer.write_f64(value);
}

pub fn write_c_string(buffer: &mut ByteBuffer, value: &str) {
    buffer.write_c_string(value);
}

pub fn write_string(buffer: &mut ByteBuffer, value: &str) {
    buffer.write_string(value);
}

pub fn write_bits(buffer: &mut ByteBuffer, value: u64, count: usize) -> Result<()> {
    buffer.write_bits(value, count)
}

pub fn write_gid(buffer: &mut ByteBuffer, gid: Gid) {
    buffer.write_u64(gid.value);
}

fn to_hex(bytes: &[u8]) -> String {
    const HEX: &[u8; 16] = b"0123456789abcdef";
    let mut output = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        output.push(HEX[(byte >> 4) as usize] as char);
        output.push(HEX[(byte & 0x0f) as usize] as char);
    }
    output
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn basic_serializer_round_trip() {
        let mut buffer = ByteBuffer::new();
        write_i8(&mut buffer, -8);
        write_u8(&mut buffer, 200);
        write_i16(&mut buffer, -1234);
        write_u16(&mut buffer, 54321);
        write_i32(&mut buffer, -555666777);
        write_u32(&mut buffer, 3332221110);
        write_i64(&mut buffer, -987654321012345);
        write_u64(&mut buffer, 111222333444555666);
        write_f32(&mut buffer, 123.5);
        write_f64(&mut buffer, 98765.4321);
        write_bool(&mut buffer, true);
        write_bool(&mut buffer, false);
        write_c_string(&mut buffer, "hello");
        write_string(&mut buffer, "world");
        write_bits(&mut buffer, 0b1101, 4).unwrap();
        write_gid(&mut buffer, Gid { value: 0x1234 });

        let raw = buffer.bytes();
        let mut reader = ByteBuffer::from_bytes(&raw);
        assert_eq!(read_i8(&mut reader).unwrap(), -8);
        assert_eq!(read_u8(&mut reader).unwrap(), 200);
        assert_eq!(read_i16(&mut reader).unwrap(), -1234);
        assert_eq!(read_u16(&mut reader).unwrap(), 54321);
        assert_eq!(read_i32(&mut reader).unwrap(), -555666777);
        assert_eq!(read_u32(&mut reader).unwrap(), 3332221110);
        assert_eq!(read_i64(&mut reader).unwrap(), -987654321012345);
        assert_eq!(read_u64(&mut reader).unwrap(), 111222333444555666);
        assert_eq!(read_f32(&mut reader).unwrap(), 123.5);
        assert_eq!(read_f64(&mut reader).unwrap(), 98765.4321);
        assert!(read_bool(&mut reader).unwrap());
        assert!(!read_bool(&mut reader).unwrap());
        assert_eq!(read_c_string(&mut reader).unwrap(), "hello");
        assert_eq!(read_string(&mut reader, 5).unwrap(), "world");
        assert_eq!(read_bits(&mut reader, 4).unwrap(), 0b1101);
        assert_eq!(read_gid(&mut reader).unwrap().value, 0x1234);
    }

    #[test]
    fn byte_string_helpers() {
        let bytes = ByteString::new(b"hello");
        assert!(!bytes.is_zero());
        assert_eq!(bytes.string(), "hello");
        assert_eq!(bytes.hex(), "68656c6c6f");

        let mut copy = bytes.bytes();
        copy[0] = b'H';
        assert_eq!(bytes.string(), "hello");
    }

    #[test]
    fn gid_helpers() {
        let gid = Gid {
            value: 0xabcdef1234567890,
        };
        assert!(!gid.is_zero());
        assert_eq!(gid.to_string(), "0xABCDEF1234567890");
        assert_eq!(gid.to_string().parse::<Gid>().unwrap(), gid);
        assert_eq!("0x10".parse::<Gid>().unwrap().value, 0x10);
    }
}
