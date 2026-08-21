# O101

O101 is a Rust implementation of the deprecated **o101** tooling. It provides a tested patching workflow for Wizard101-style content deployments together with protocol utilities, WAD archive support, message capture sorting, and an experimental multiplayer realm emulator.

## Features

- **Concurrent patching** — bounded parallel downloads with size and SHA-256 validation.
- **Strict manifests** — validated JSON manifests with traversal-safe install paths.
- **Crash-resistant state** — deterministic JSON state persisted through temporary-file replacement.
- **Three CLI tools** — `wizturtle`, `wizserver`, and `messagesorter`.
- **Open101 primitives** — little-endian/bit-packed byte buffers, WAD readers, CRC validation, zlib support, and serializer helpers.
- **Realm model** — concurrency-safe zones, character caching, broadcast handling, and serialization helpers.
- **Rust tests and CI** — formatting, Clippy, unit/integration behavior, and binary builds are verified in GitHub Actions.

## Requirements

- Rust 1.88 or newer.

## Installation

```bash
cargo install --git https://github.com/cybellereaper/o101 --bin wizturtle
cargo install --git https://github.com/cybellereaper/o101 --bin wizserver
cargo install --git https://github.com/cybellereaper/o101 --bin messagesorter
```

## Patcher

```bash
wizturtle \
  --patch-info https://example.com/patch-info.json \
  --install-dir /path/to/install \
  --state-file /path/to/install/.wizturtle/state.json
```

The patch info endpoint is expected to respond with JSON similar to:

```json
{
  "version": "1.0.0",
  "manifest": "https://example.com/manifest.json",
  "base_url": "https://example.com/"
}
```

The referenced manifest contains the files to install:

```json
{
  "version": "1.0.0",
  "files": [
    {
      "src": "files/Bin/app.bin",
      "dst": "Bin/app.bin",
      "size": 1024,
      "sha256": "<64 hex characters>",
      "mode": "0644"
    }
  ]
}
```

## Realm emulator

```bash
wizserver \
  --game-dir /path/to/game \
  --login-addr 127.0.0.1:12500 \
  --game-addr 127.0.0.1:12501 \
  --max-players 100 \
  --zones 50 \
  --zone-capacity 10
```

The TCP services send an informational greeting and echo the first payload received. The `o101::wizserver` module exposes reusable `Realm`, `Zone`, `InGameCharacter`, and serialization helpers for protocol experiments.

## Open101 IO and serialization

`o101::open101::bytebuffer` provides little-endian scalar encoding and the original Open101 bit-packing behavior. `o101::open101::wad` parses KIWAD archives, validates CRC32, handles compressed entries, and caches WADs through `Manager`. `o101::open101::serializer` provides `ByteString`, `Gid`, and strongly typed serializer helpers.

## Message capture sorting

```bash
messagesorter /path/to/MessageSorter/Traffic/ServiceCapture.xml
# wrote 56 messages for service LoginService (42) to ./42_LoginService.txt
```

Use `--out <directory>` to select another output directory. Output follows the `<ServiceID>_<ServiceName>.txt` naming convention.

## Development

```bash
cargo fmt --all -- --check
cargo clippy --all-targets --all-features -- -D warnings
cargo test --all-targets
cargo build --release --bins
```

## License

GNU General Public License v3.0
