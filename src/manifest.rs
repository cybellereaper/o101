use reqwest::Url;
use serde::Deserialize;

use crate::{Error, Result};

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct Manifest {
    pub version: String,
    pub files: Vec<File>,
}

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct File {
    pub source: String,
    pub target: String,
    pub size: u64,
    pub sha256: String,
    pub mode: Option<u32>,
}

#[derive(Deserialize)]
struct RawManifest {
    version: String,
    files: Vec<RawFile>,
}

#[derive(Deserialize)]
struct RawFile {
    #[serde(rename = "src")]
    source: String,
    #[serde(rename = "dst")]
    target: String,
    size: i64,
    sha256: String,
    mode: Option<String>,
}

pub fn parse(data: &[u8]) -> Result<Manifest> {
    let raw: RawManifest = serde_json::from_slice(data)
        .map_err(|error| Error::message(format!("manifest: decode: {error}")))?;

    if raw.version.trim().is_empty() {
        return Err(Error::message("manifest: version is required"));
    }
    if raw.files.is_empty() {
        return Err(Error::message("manifest: files list is empty"));
    }

    let files = raw
        .files
        .into_iter()
        .enumerate()
        .map(|(index, file)| {
            decode_file(file)
                .map_err(|error| Error::message(format!("manifest: files[{index}]: {error}")))
        })
        .collect::<Result<Vec<_>>>()?;

    Ok(Manifest {
        version: raw.version,
        files,
    })
}

fn decode_file(raw: RawFile) -> Result<File> {
    let source = normalize_source(&raw.source)?;
    let target = normalize_target(&raw.target)?;

    if raw.size <= 0 {
        return Err(Error::message("decode: size must be positive"));
    }
    if raw.sha256.len() != 64 || !raw.sha256.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        return Err(Error::message("decode: sha256 must be 64 hex characters"));
    }

    let mode = raw
        .mode
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(parse_file_mode)
        .transpose()?;

    Ok(File {
        source,
        target,
        size: raw.size as u64,
        sha256: raw.sha256.to_ascii_lowercase(),
        mode,
    })
}

fn normalize_source(value: &str) -> Result<String> {
    let value = value.trim();
    if value.is_empty() {
        return Err(Error::message("decode: src is required"));
    }

    if Url::parse(value).is_ok() {
        return Ok(value.to_owned());
    }

    normalize_relative(value, "src")
}

fn normalize_target(value: &str) -> Result<String> {
    let value = value.trim();
    if value.is_empty() {
        return Err(Error::message("decode: dst is required"));
    }
    if value.contains('\\') {
        return Err(Error::message("decode: dst must use forward slashes"));
    }

    normalize_relative(value, "dst")
}

fn normalize_relative(value: &str, field: &str) -> Result<String> {
    if value.starts_with('/') {
        return Err(Error::message(format!("decode: {field} must be relative")));
    }

    let mut parts = Vec::new();
    for part in value.split('/') {
        match part {
            "" | "." => {}
            ".." => {
                if parts.pop().is_none() {
                    return Err(Error::message(format!(
                        "decode: {field} must not escape its root"
                    )));
                }
            }
            part => parts.push(part),
        }
    }

    if parts.is_empty() {
        return Err(Error::message(format!("decode: {field} is required")));
    }

    Ok(parts.join("/"))
}

fn parse_file_mode(value: &str) -> Result<u32> {
    if value.is_empty() {
        return Err(Error::message("decode: mode: empty mode"));
    }
    if !value.bytes().all(|byte| matches!(byte, b'0'..=b'7')) {
        return Err(Error::message(format!(
            "decode: mode: invalid octal mode {value:?}"
        )));
    }

    u32::from_str_radix(value, 8).map_err(|error| Error::message(format!("decode: mode: {error}")))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_valid_manifest() {
        let raw = br#"{
            "version":"1.2.3",
            "files":[{
                "src":"assets/app.bin",
                "dst":"Bin/app.bin",
                "size":42,
                "sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                "mode":"0644"
            }]
        }"#;

        let manifest = parse(raw).expect("valid manifest");
        assert_eq!(manifest.version, "1.2.3");
        assert_eq!(manifest.files.len(), 1);
        assert_eq!(manifest.files[0].source, "assets/app.bin");
        assert_eq!(manifest.files[0].target, "Bin/app.bin");
        assert_eq!(manifest.files[0].mode, Some(0o644));
    }

    #[test]
    fn rejects_invalid_entries() {
        let raw = br#"{
            "version":"1.0.0",
            "files":[{"src":"","dst":"file","size":1,"sha256":""}]
        }"#;
        assert!(parse(raw).is_err());
    }

    #[test]
    fn rejects_target_traversal() {
        let raw = br#"{
            "version":"1.0.0",
            "files":[{
                "src":"file.bin",
                "dst":"../outside.bin",
                "size":1,
                "sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
            }]
        }"#;
        assert!(parse(raw).is_err());
    }
}
