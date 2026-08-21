use std::{
    collections::HashMap,
    fs::{self, File},
    io::{Read, Seek, SeekFrom},
    path::{Path, PathBuf},
    sync::{Arc, RwLock},
};

use flate2::read::ZlibDecoder;

use crate::{Error, Result};

const WAD_MAGIC: &[u8; 5] = b"KIWAD";

#[derive(Clone, Debug, Eq, PartialEq)]
pub struct FileRecord {
    pub offset: u32,
    pub size: u32,
    pub compressed_size: u32,
    pub compressed: bool,
    pub crc: u32,
    pub name: String,
}

#[derive(Debug)]
pub struct Wad {
    path: PathBuf,
    records: Vec<FileRecord>,
    index: HashMap<String, FileRecord>,
}

impl Wad {
    pub fn open(path: impl Into<PathBuf>) -> Result<Self> {
        let path = path.into();
        let mut file = File::open(&path)?;
        let records = read_header(&mut file)?;
        let index = records
            .iter()
            .filter(|record| !record.name.is_empty())
            .map(|record| (record.name.clone(), record.clone()))
            .collect();

        Ok(Self {
            path,
            records,
            index,
        })
    }

    pub fn records(&self) -> &[FileRecord] {
        &self.records
    }

    pub fn open_file(&self, name: &str) -> Result<Vec<u8>> {
        let record = self
            .index
            .get(name)
            .ok_or_else(|| Error::message("wad: file not found"))?;

        let mut file = File::open(&self.path)?;
        file.seek(SeekFrom::Start(u64::from(record.offset)))?;

        let chunk_size = if record.compressed_size == 0 {
            record.size
        } else {
            record.compressed_size
        };
        let mut raw = vec![0_u8; chunk_size as usize];
        file.read_exact(&mut raw)?;

        let data = if record.compressed {
            let mut decoder = ZlibDecoder::new(raw.as_slice());
            let mut data = Vec::with_capacity(record.size as usize);
            decoder.read_to_end(&mut data)?;
            if data.len() != record.size as usize {
                return Err(Error::message(format!(
                    "wad: size mismatch for {name}: expected {}, got {}",
                    record.size,
                    data.len()
                )));
            }
            data
        } else {
            if raw.len() < record.size as usize {
                return Err(Error::message(format!(
                    "wad: entry {name} is shorter than declared size"
                )));
            }
            raw[..record.size as usize].to_vec()
        };

        if crc32fast::hash(&data) != record.crc {
            return Err(Error::message("wad: crc mismatch"));
        }

        Ok(data)
    }
}

pub fn read_header<R: Read>(reader: &mut R) -> Result<Vec<FileRecord>> {
    let mut magic = [0_u8; 5];
    reader
        .read_exact(&mut magic)
        .map_err(|error| match error.kind() {
            std::io::ErrorKind::UnexpectedEof => Error::message("wad: unexpected end of header"),
            _ => error.into(),
        })?;
    if &magic != WAD_MAGIC {
        return Err(Error::message("wad: invalid header"));
    }

    let version = read_u32(reader)?;
    let file_count = read_u32(reader)?;
    if version >= 2 {
        read_u8(reader)?;
    }

    let mut records = Vec::with_capacity(file_count as usize);
    for _ in 0..file_count {
        let offset = read_u32(reader)?;
        let size = read_u32(reader)?;
        let compressed_size = read_u32(reader)?;
        let compressed = read_u8(reader)? != 0;
        let crc = read_u32(reader)?;
        let name_length = read_i32(reader)?;

        let name = if name_length <= 0 {
            String::new()
        } else {
            let length = usize::try_from(name_length - 1)
                .map_err(|_| Error::message("wad: invalid name length"))?;
            let mut name = vec![0_u8; length];
            reader.read_exact(&mut name)?;
            if read_u8(reader)? != 0 {
                return Err(Error::message("wad: expected null terminator"));
            }
            String::from_utf8_lossy(&name).into_owned()
        };

        records.push(FileRecord {
            offset,
            size,
            compressed_size,
            compressed,
            crc,
            name,
        });
    }

    Ok(records)
}

#[derive(Debug)]
pub struct Manager {
    inner: RwLock<ManagerState>,
}

#[derive(Debug)]
struct ManagerState {
    game_dir: PathBuf,
    data_dir: PathBuf,
    wads: HashMap<String, Arc<Wad>>,
}

impl Manager {
    pub fn new(game_dir: impl AsRef<Path>) -> Self {
        let game_dir = game_dir.as_ref().to_path_buf();
        let data_dir = if game_dir.as_os_str().is_empty() {
            PathBuf::new()
        } else {
            game_dir.join("Data").join("GameData")
        };

        Self {
            inner: RwLock::new(ManagerState {
                game_dir,
                data_dir,
                wads: HashMap::new(),
            }),
        }
    }

    pub fn set_game_dir(&self, directory: impl AsRef<Path>) -> Result<()> {
        let directory = directory.as_ref().to_path_buf();
        let data_dir = if directory.as_os_str().is_empty() {
            PathBuf::new()
        } else {
            directory.join("Data").join("GameData")
        };
        let mut state = self.write_state()?;
        state.game_dir = directory;
        state.data_dir = data_dir;
        state.wads.clear();
        Ok(())
    }

    pub fn set_game_data_dir(&self, directory: impl AsRef<Path>) -> Result<()> {
        let mut state = self.write_state()?;
        state.game_dir.clear();
        state.data_dir = directory.as_ref().to_path_buf();
        state.wads.clear();
        Ok(())
    }

    pub fn game_dir(&self) -> Result<PathBuf> {
        Ok(self.read_state()?.game_dir.clone())
    }

    pub fn data_dir(&self) -> Result<PathBuf> {
        Ok(self.read_state()?.data_dir.clone())
    }

    pub fn open_file(&self, name: &str) -> Result<Vec<u8>> {
        let (wad_name, file_name) = parse_resource_name(name);
        self.get_wad(&wad_name)?.open_file(&file_name)
    }

    pub fn get_file_bytes(&self, name: &str) -> Result<Vec<u8>> {
        self.open_file(name)
    }

    pub fn dump_file(&self, name: &str) -> Result<PathBuf> {
        let data = self.get_file_bytes(name)?;
        let (_, file_name) = parse_resource_name(name);
        let path = PathBuf::from(file_name);
        fs::write(&path, data)?;
        Ok(path)
    }

    pub fn add_custom_wad(&self, path: impl AsRef<Path>) -> Result<Arc<Wad>> {
        let path = path.as_ref();
        let wad = Arc::new(Wad::open(path)?);
        let name = path
            .file_stem()
            .and_then(|value| value.to_str())
            .ok_or_else(|| Error::message("wad: custom wad has no valid file name"))?
            .to_owned();
        self.write_state()?.wads.insert(name, Arc::clone(&wad));
        Ok(wad)
    }

    pub fn cached_wad(&self, name: &str) -> Result<Option<Arc<Wad>>> {
        Ok(self.read_state()?.wads.get(name).cloned())
    }

    fn get_wad(&self, name: &str) -> Result<Arc<Wad>> {
        if let Some(wad) = self.read_state()?.wads.get(name).cloned() {
            return Ok(wad);
        }

        let mut state = self.write_state()?;
        if let Some(wad) = state.wads.get(name).cloned() {
            return Ok(wad);
        }
        if state.data_dir.as_os_str().is_empty() {
            return Err(Error::message("wad: data directory not configured"));
        }

        let wad = Arc::new(Wad::open(state.data_dir.join(format!("{name}.wad")))?);
        state.wads.insert(name.to_owned(), Arc::clone(&wad));
        Ok(wad)
    }

    fn read_state(&self) -> Result<std::sync::RwLockReadGuard<'_, ManagerState>> {
        self.inner
            .read()
            .map_err(|_| Error::message("wad: manager lock poisoned"))
    }

    fn write_state(&self) -> Result<std::sync::RwLockWriteGuard<'_, ManagerState>> {
        self.inner
            .write()
            .map_err(|_| Error::message("wad: manager lock poisoned"))
    }
}

fn parse_resource_name(name: &str) -> (String, String) {
    let mut wad_name = "Root".to_owned();
    let mut file_name = name;

    if let Some(rest) = name.strip_prefix('|')
        && let Some(last) = rest.rfind('|')
    {
        wad_name = rest[..last].replace('|', "-");
        file_name = &rest[last + 1..];
    }

    (wad_name, file_name.replace('\\', "/"))
}

fn read_u8<R: Read>(reader: &mut R) -> Result<u8> {
    let mut bytes = [0_u8; 1];
    reader.read_exact(&mut bytes)?;
    Ok(bytes[0])
}

fn read_u32<R: Read>(reader: &mut R) -> Result<u32> {
    let mut bytes = [0_u8; 4];
    reader.read_exact(&mut bytes)?;
    Ok(u32::from_le_bytes(bytes))
}

fn read_i32<R: Read>(reader: &mut R) -> Result<i32> {
    let mut bytes = [0_u8; 4];
    reader.read_exact(&mut bytes)?;
    Ok(i32::from_le_bytes(bytes))
}

#[cfg(test)]
mod tests {
    use std::io::Write;

    use flate2::{Compression, write::ZlibEncoder};

    use super::*;
    use crate::test_support::TempDir;

    struct WadEntry<'a> {
        name: &'a str,
        data: &'a [u8],
        compressed: bool,
    }

    fn build_test_wad(entries: &[WadEntry<'_>]) -> Vec<u8> {
        let version = 2_u32;
        let mut header_size = 5 + 4 + 4 + 1;
        for entry in entries {
            header_size += 4 + 4 + 4 + 1 + 4 + 4 + entry.name.len() + 1;
        }

        let mut header = Vec::with_capacity(header_size);
        header.extend_from_slice(WAD_MAGIC);
        header.extend_from_slice(&version.to_le_bytes());
        header.extend_from_slice(&(entries.len() as u32).to_le_bytes());
        header.push(0);

        let mut data_section = Vec::new();
        let mut offset = header_size as u32;
        for entry in entries {
            let stored = if entry.compressed {
                let mut encoder = ZlibEncoder::new(Vec::new(), Compression::default());
                encoder.write_all(entry.data).unwrap();
                encoder.finish().unwrap()
            } else {
                entry.data.to_vec()
            };

            header.extend_from_slice(&offset.to_le_bytes());
            header.extend_from_slice(&(entry.data.len() as u32).to_le_bytes());
            header.extend_from_slice(&(stored.len() as u32).to_le_bytes());
            header.push(u8::from(entry.compressed));
            header.extend_from_slice(&crc32fast::hash(entry.data).to_le_bytes());
            header.extend_from_slice(&((entry.name.len() + 1) as i32).to_le_bytes());
            header.extend_from_slice(entry.name.as_bytes());
            header.push(0);

            data_section.extend_from_slice(&stored);
            offset += stored.len() as u32;
        }

        header.extend_from_slice(&data_section);
        header
    }

    #[test]
    fn reads_header() {
        let bytes = build_test_wad(&[WadEntry {
            name: "foo/bar.bin",
            data: b"hello",
            compressed: false,
        }]);
        let mut reader = bytes.as_slice();
        let records = read_header(&mut reader).expect("header");
        assert_eq!(records.len(), 1);
        assert_eq!(records[0].name, "foo/bar.bin");
        assert_eq!(records[0].size, 5);
        assert_ne!(records[0].offset, 0);
    }

    #[test]
    fn opens_plain_and_compressed_files() {
        let entries = [
            WadEntry {
                name: "plain.txt",
                data: b"plain",
                compressed: false,
            },
            WadEntry {
                name: "compressed.bin",
                data: b"compress me",
                compressed: true,
            },
        ];
        let dir = TempDir::new("wad");
        let path = dir.path().join("Root.wad");
        fs::write(&path, build_test_wad(&entries)).unwrap();
        let wad = Wad::open(&path).unwrap();

        for entry in &entries {
            assert_eq!(wad.open_file(entry.name).unwrap(), entry.data);
        }
        assert!(wad.open_file("missing").is_err());
    }

    #[test]
    fn manager_lifecycle() {
        let entries = [WadEntry {
            name: "foo.bin",
            data: b"abc",
            compressed: true,
        }];
        let bytes = build_test_wad(&entries);
        let dir = TempDir::new("wad-manager");
        let data_dir = dir.path().join("Data").join("GameData");
        fs::create_dir_all(&data_dir).unwrap();
        fs::write(data_dir.join("Root.wad"), &bytes).unwrap();

        let manager = Manager::new(dir.path());
        assert_eq!(manager.get_file_bytes("foo.bin").unwrap(), b"abc");

        let custom = dir.path().join("custom.wad");
        fs::write(&custom, bytes).unwrap();
        manager.add_custom_wad(&custom).unwrap();
        assert_eq!(manager.open_file("|custom|foo.bin").unwrap(), b"abc");
    }
}
