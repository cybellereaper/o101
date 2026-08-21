use std::{
    collections::BTreeMap,
    fs::{self, File, OpenOptions},
    io::{Read, Write},
    path::{Path, PathBuf},
    sync::{
        Arc,
        atomic::{AtomicBool, AtomicU64, Ordering},
    },
    thread,
    time::Duration,
};

use reqwest::{StatusCode, Url, blocking::Client};
use serde::Deserialize;
use sha2::{Digest, Sha256};

use crate::{
    Error, Result,
    manifest::{self, File as ManifestFile, Manifest},
    state::{FileInfo, Snapshot, Store},
};

static TEMP_COUNTER: AtomicU64 = AtomicU64::new(0);

pub trait Logger: Send + Sync {
    fn log(&self, message: &str);
}

impl<F> Logger for F
where
    F: Fn(&str) + Send + Sync,
{
    fn log(&self, message: &str) {
        self(message);
    }
}

pub struct Config {
    pub patch_info_url: String,
    pub install_dir: PathBuf,
    pub state_store: Arc<Store>,
    pub http_client: Option<Client>,
    pub concurrency: usize,
    pub logger: Option<Arc<dyn Logger>>,
}

#[derive(Clone, Debug, Deserialize)]
struct PatchInfo {
    version: String,
    #[serde(rename = "manifest")]
    manifest_url: String,
    #[serde(default)]
    base_url: String,
}

#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum PatchOutcome {
    Updated,
    UpToDate,
}

pub struct Patcher {
    patch_info_url: String,
    install_dir: PathBuf,
    state_store: Arc<Store>,
    client: Client,
    concurrency: usize,
    logger: Option<Arc<dyn Logger>>,
}

impl Patcher {
    pub fn new(config: Config) -> Result<Self> {
        if config.patch_info_url.trim().is_empty() {
            return Err(Error::message("patcher: patch info URL is required"));
        }
        if config.install_dir.as_os_str().is_empty() {
            return Err(Error::message("patcher: install directory is required"));
        }

        let client = match config.http_client {
            Some(client) => client,
            None => Client::builder().timeout(Duration::from_secs(30)).build()?,
        };
        let concurrency = if config.concurrency == 0 {
            thread::available_parallelism()
                .map(usize::from)
                .unwrap_or(2)
                .max(2)
        } else {
            config.concurrency
        };

        Ok(Self {
            patch_info_url: config.patch_info_url,
            install_dir: config.install_dir,
            state_store: config.state_store,
            client,
            concurrency,
            logger: config.logger,
        })
    }

    pub fn run(&self, cancelled: &AtomicBool) -> Result<PatchOutcome> {
        let previous = self.state_store.load()?;
        self.check_cancelled(cancelled)?;

        let info = self.fetch_patch_info()?;
        self.check_cancelled(cancelled)?;

        let manifest_bytes = self
            .download(&info.manifest_url)
            .map_err(|error| Error::message(format!("patcher: manifest download: {error}")))?;
        let manifest = manifest::parse(&manifest_bytes)?;
        let base_url = manifest_base_url(&info)?;

        let mut pending = Vec::new();
        let mut next_state = Snapshot {
            version: manifest.version.clone(),
            files: BTreeMap::new(),
        };

        for entry in &manifest.files {
            self.check_cancelled(cancelled)?;
            let destination = self.destination(entry);
            match validate_local_file(&destination, entry) {
                Ok(Some(metadata)) => {
                    next_state.files.insert(entry.target.clone(), metadata);
                }
                Ok(None) => pending.push(entry.clone()),
                Err(error) => {
                    self.log(&format!("invalid file {}: {error}", entry.target));
                    pending.push(entry.clone());
                }
            }
        }

        if pending.is_empty() && previous.version == manifest.version {
            self.state_store.save(&next_state)?;
            return Ok(PatchOutcome::UpToDate);
        }

        self.log(&format!(
            "Starting patch to version {} ({} files to update)",
            manifest.version,
            pending.len()
        ));

        for (target, metadata) in self.download_entries(&base_url, &pending, cancelled)? {
            next_state.files.insert(target, metadata);
        }

        fill_missing_metadata(self, &manifest, &mut next_state)?;
        self.state_store.save(&next_state)?;
        self.log("Patch completed successfully");
        Ok(PatchOutcome::Updated)
    }

    fn fetch_patch_info(&self) -> Result<PatchInfo> {
        let body = self
            .download(&self.patch_info_url)
            .map_err(|error| Error::message(format!("patcher: patch info download: {error}")))?;
        let info: PatchInfo = serde_json::from_slice(&body)
            .map_err(|error| Error::message(format!("patcher: decode patch info: {error}")))?;

        if info.version.trim().is_empty() {
            return Err(Error::message("patcher: patch info missing version"));
        }
        if info.manifest_url.trim().is_empty() {
            return Err(Error::message("patcher: patch info missing manifest URL"));
        }
        Ok(info)
    }

    fn download(&self, resource: &str) -> Result<Vec<u8>> {
        let response = self.client.get(resource).send()?;
        if response.status() != StatusCode::OK {
            return Err(Error::message(format!(
                "unexpected status {} from {resource}",
                response.status()
            )));
        }
        Ok(response.bytes()?.to_vec())
    }

    fn download_entries(
        &self,
        base_url: &Url,
        entries: &[ManifestFile],
        cancelled: &AtomicBool,
    ) -> Result<Vec<(String, FileInfo)>> {
        let mut output = Vec::with_capacity(entries.len());

        for batch in entries.chunks(self.concurrency) {
            self.check_cancelled(cancelled)?;
            let results = thread::scope(|scope| {
                let mut handles = Vec::with_capacity(batch.len());
                for entry in batch.iter().cloned() {
                    handles.push(scope.spawn(move || {
                        let metadata = self.download_entry(base_url, &entry, cancelled)?;
                        Ok::<_, Error>((entry.target, metadata))
                    }));
                }

                let mut results = Vec::with_capacity(handles.len());
                for handle in handles {
                    let result = handle
                        .join()
                        .map_err(|_| Error::message("patcher: download worker panicked"))??;
                    results.push(result);
                }
                Ok::<_, Error>(results)
            })?;
            output.extend(results);
        }

        Ok(output)
    }

    fn download_entry(
        &self,
        base_url: &Url,
        entry: &ManifestFile,
        cancelled: &AtomicBool,
    ) -> Result<FileInfo> {
        let destination = self.destination(entry);
        if let Some(parent) = destination.parent() {
            fs::create_dir_all(parent)?;
        }

        let target_url = resolve_url(base_url, &entry.source)?;
        let mut response = self.client.get(target_url.clone()).send()?;
        if response.status() != StatusCode::OK {
            return Err(Error::message(format!(
                "download {}: status {}",
                entry.source,
                response.status()
            )));
        }

        let directory = destination.parent().unwrap_or_else(|| Path::new("."));
        let (temp_path, mut temp_file) = create_temp_file(directory)?;
        let result = (|| {
            let mut hasher = Sha256::new();
            let mut written = 0_u64;
            let mut buffer = [0_u8; 64 * 1024];

            loop {
                self.check_cancelled(cancelled)?;
                let read = response.read(&mut buffer)?;
                if read == 0 {
                    break;
                }
                temp_file.write_all(&buffer[..read])?;
                hasher.update(&buffer[..read]);
                written += read as u64;
            }

            if written != entry.size {
                return Err(Error::message(format!(
                    "size mismatch for {}: expected {}, got {written}",
                    entry.target, entry.size
                )));
            }

            let digest = to_hex(&hasher.finalize());
            if digest != entry.sha256 {
                return Err(Error::message(format!(
                    "hash mismatch for {}",
                    entry.target
                )));
            }

            temp_file.sync_all()?;
            drop(temp_file);
            replace_file(&temp_path, &destination)?;
            apply_mode(&destination, entry.mode)?;

            Ok(FileInfo {
                size: written,
                sha256: digest,
            })
        })();

        if result.is_err() {
            let _ = fs::remove_file(&temp_path);
        }
        result
    }

    fn destination(&self, entry: &ManifestFile) -> PathBuf {
        entry
            .target
            .split('/')
            .fold(self.install_dir.clone(), |path, component| {
                path.join(component)
            })
    }

    fn check_cancelled(&self, cancelled: &AtomicBool) -> Result<()> {
        if cancelled.load(Ordering::Relaxed) {
            Err(Error::message("patcher: cancelled"))
        } else {
            Ok(())
        }
    }

    fn log(&self, message: &str) {
        if let Some(logger) = &self.logger {
            logger.log(message);
        }
    }
}

fn fill_missing_metadata(
    patcher: &Patcher,
    manifest: &Manifest,
    snapshot: &mut Snapshot,
) -> Result<()> {
    for entry in &manifest.files {
        if snapshot.files.contains_key(&entry.target) {
            continue;
        }
        let destination = patcher.destination(entry);
        let metadata = validate_local_file(&destination, entry)?.ok_or_else(|| {
            Error::message(format!(
                "patcher: expected {} to be valid after download",
                entry.target
            ))
        })?;
        snapshot.files.insert(entry.target.clone(), metadata);
    }
    Ok(())
}

fn manifest_base_url(info: &PatchInfo) -> Result<Url> {
    let value = if info.base_url.trim().is_empty() {
        &info.manifest_url
    } else {
        &info.base_url
    };
    Url::parse(value).map_err(|error| Error::message(format!("patcher: invalid base URL: {error}")))
}

fn resolve_url(base_url: &Url, source: &str) -> Result<Url> {
    if let Ok(url) = Url::parse(source) {
        return Ok(url);
    }
    base_url
        .join(source)
        .map_err(|error| Error::message(format!("resolve {source}: {error}")))
}

fn validate_local_file(path: &Path, entry: &ManifestFile) -> Result<Option<FileInfo>> {
    let metadata = match fs::metadata(path) {
        Ok(metadata) => metadata,
        Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
        Err(error) => return Err(error.into()),
    };

    if metadata.len() != entry.size {
        return Ok(None);
    }

    let mut file = File::open(path)?;
    let mut hasher = Sha256::new();
    let mut buffer = [0_u8; 64 * 1024];
    loop {
        let read = file.read(&mut buffer)?;
        if read == 0 {
            break;
        }
        hasher.update(&buffer[..read]);
    }

    let digest = to_hex(&hasher.finalize());
    if digest != entry.sha256 {
        return Ok(None);
    }

    Ok(Some(FileInfo {
        size: entry.size,
        sha256: digest,
    }))
}

fn create_temp_file(directory: &Path) -> Result<(PathBuf, File)> {
    for _ in 0..32 {
        let counter = TEMP_COUNTER.fetch_add(1, Ordering::Relaxed);
        let path = directory.join(format!(".wizturtle-{}-{counter}.tmp", std::process::id()));
        match OpenOptions::new().write(true).create_new(true).open(&path) {
            Ok(file) => return Ok((path, file)),
            Err(error) if error.kind() == std::io::ErrorKind::AlreadyExists => continue,
            Err(error) => return Err(error.into()),
        }
    }
    Err(Error::message("patcher: unable to allocate temporary file"))
}

fn replace_file(source: &Path, destination: &Path) -> std::io::Result<()> {
    #[cfg(windows)]
    if destination.exists() {
        fs::remove_file(destination)?;
    }

    fs::rename(source, destination)
}

#[cfg(unix)]
fn apply_mode(path: &Path, mode: Option<u32>) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;

    if let Some(mode) = mode {
        fs::set_permissions(path, fs::Permissions::from_mode(mode))?;
    }
    Ok(())
}

#[cfg(not(unix))]
fn apply_mode(_path: &Path, _mode: Option<u32>) -> Result<()> {
    Ok(())
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
    use std::{collections::HashMap, sync::OnceLock};

    use serde_json::json;

    use super::*;
    use crate::test_support::{TempDir, TestResponse, TestServer};

    fn test_server(version: &str, files: HashMap<String, Vec<u8>>) -> TestServer {
        let version = version.to_owned();
        let base_url = Arc::new(OnceLock::<String>::new());
        let handler_base_url = Arc::clone(&base_url);
        let files = Arc::new(files);
        let handler_files = Arc::clone(&files);

        let manifest_files = files
            .iter()
            .map(|(name, data)| {
                let mut hasher = Sha256::new();
                hasher.update(data);
                json!({
                    "src": format!("files/{name}"),
                    "dst": name,
                    "size": data.len(),
                    "sha256": to_hex(&hasher.finalize()),
                })
            })
            .collect::<Vec<_>>();
        let manifest = serde_json::to_vec(&json!({
            "version": version,
            "files": manifest_files,
        }))
        .unwrap();

        let handler_version = version.clone();
        let server = TestServer::new(move |path| match path {
            "/patch-info" => TestResponse::ok_json(
                serde_json::to_vec(&json!({
                    "version": handler_version,
                    "manifest": format!("{}/manifest.json", handler_base_url.get().unwrap()),
                }))
                .unwrap(),
            ),
            "/manifest.json" => TestResponse::ok_json(manifest.clone()),
            path if path.starts_with("/files/") => {
                let name = &path[7..];
                handler_files
                    .get(name)
                    .cloned()
                    .map(TestResponse::ok_bytes)
                    .unwrap_or_else(TestResponse::not_found)
            }
            _ => TestResponse::not_found(),
        });
        base_url.set(server.url("")).unwrap();
        server
    }

    fn patcher(server: &TestServer, directory: &TempDir) -> Patcher {
        Patcher::new(Config {
            patch_info_url: server.url("/patch-info"),
            install_dir: directory.path().to_path_buf(),
            state_store: Arc::new(Store::new(directory.path().join("state.json"))),
            http_client: None,
            concurrency: 2,
            logger: None,
        })
        .unwrap()
    }

    #[test]
    fn downloads_missing_files_and_skips_when_current() {
        let files = HashMap::from([("Bin/app.bin".to_owned(), b"hello world".to_vec())]);
        let server = test_server("1.0.0", files);
        let directory = TempDir::new("patcher");
        let patcher = patcher(&server, &directory);
        let cancelled = AtomicBool::new(false);

        assert_eq!(patcher.run(&cancelled).unwrap(), PatchOutcome::Updated);
        assert_eq!(
            fs::read(directory.path().join("Bin/app.bin")).unwrap(),
            b"hello world"
        );
        assert_eq!(patcher.run(&cancelled).unwrap(), PatchOutcome::UpToDate);
    }

    #[test]
    fn repairs_corrupted_files() {
        let files = HashMap::from([("Bin/app.bin".to_owned(), b"hello world".to_vec())]);
        let server = test_server("1.0.0", files);
        let directory = TempDir::new("patcher-corruption");
        let patcher = patcher(&server, &directory);
        let cancelled = AtomicBool::new(false);

        patcher.run(&cancelled).unwrap();
        fs::write(directory.path().join("Bin/app.bin"), b"tampered").unwrap();
        assert_eq!(patcher.run(&cancelled).unwrap(), PatchOutcome::Updated);
        assert_eq!(
            fs::read(directory.path().join("Bin/app.bin")).unwrap(),
            b"hello world"
        );
    }
}
