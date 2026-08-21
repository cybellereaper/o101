use std::{
    collections::BTreeMap,
    fs::{self, File},
    io::Write,
    path::{Path, PathBuf},
    sync::Mutex,
};

use serde::{Deserialize, Serialize};

use crate::{Error, Result};

#[derive(Clone, Debug, Default, Deserialize, Eq, PartialEq, Serialize)]
pub struct Snapshot {
    #[serde(default)]
    pub version: String,
    #[serde(default)]
    pub files: BTreeMap<String, FileInfo>,
}

#[derive(Clone, Debug, Deserialize, Eq, PartialEq, Serialize)]
pub struct FileInfo {
    pub size: u64,
    pub sha256: String,
}

#[derive(Debug)]
pub struct Store {
    path: PathBuf,
    lock: Mutex<()>,
}

impl Store {
    pub fn new(path: impl Into<PathBuf>) -> Self {
        Self {
            path: path.into(),
            lock: Mutex::new(()),
        }
    }

    pub fn path(&self) -> &Path {
        &self.path
    }

    pub fn load(&self) -> Result<Snapshot> {
        let _guard = self
            .lock
            .lock()
            .map_err(|_| Error::message("state: lock poisoned"))?;

        self.validate_path()?;

        let data = match fs::read(&self.path) {
            Ok(data) => data,
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => {
                return Ok(Snapshot::default());
            }
            Err(error) => return Err(Error::message(format!("state: read: {error}"))),
        };

        serde_json::from_slice(&data)
            .map_err(|error| Error::message(format!("state: decode: {error}")))
    }

    pub fn save(&self, snapshot: &Snapshot) -> Result<()> {
        let _guard = self
            .lock
            .lock()
            .map_err(|_| Error::message("state: lock poisoned"))?;

        self.validate_path()?;

        let data = serde_json::to_vec_pretty(snapshot)
            .map_err(|error| Error::message(format!("state: encode: {error}")))?;

        if let Some(parent) = self.path.parent()
            && !parent.as_os_str().is_empty()
        {
            fs::create_dir_all(parent)
                .map_err(|error| Error::message(format!("state: mkdir: {error}")))?;
        }

        let temp_path = temporary_path(&self.path);
        let mut file = File::create(&temp_path)
            .map_err(|error| Error::message(format!("state: write tmp: {error}")))?;
        file.write_all(&data)
            .map_err(|error| Error::message(format!("state: write tmp: {error}")))?;
        file.sync_all()
            .map_err(|error| Error::message(format!("state: sync tmp: {error}")))?;
        drop(file);

        replace_file(&temp_path, &self.path)
            .map_err(|error| Error::message(format!("state: rename: {error}")))?;

        Ok(())
    }

    fn validate_path(&self) -> Result<()> {
        if self.path.as_os_str().is_empty() {
            return Err(Error::message("state: path is required"));
        }
        Ok(())
    }
}

fn temporary_path(path: &Path) -> PathBuf {
    let mut value = path.as_os_str().to_os_string();
    value.push(".tmp");
    PathBuf::from(value)
}

fn replace_file(source: &Path, destination: &Path) -> std::io::Result<()> {
    #[cfg(windows)]
    if destination.exists() {
        fs::remove_file(destination)?;
    }

    fs::rename(source, destination)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_support::TempDir;

    #[test]
    fn store_loads_and_saves() {
        let dir = TempDir::new("state");
        let store = Store::new(dir.path().join("state.json"));

        let mut snapshot = store.load().expect("initial load");
        assert_eq!(snapshot, Snapshot::default());

        snapshot.version = "1.0.0".to_owned();
        snapshot.files.insert(
            "file".to_owned(),
            FileInfo {
                size: 10,
                sha256: "hash".to_owned(),
            },
        );
        store.save(&snapshot).expect("save");

        let loaded = store.load().expect("reload");
        assert_eq!(loaded.version, "1.0.0");
        assert_eq!(loaded.files["file"].size, 10);
    }
}
