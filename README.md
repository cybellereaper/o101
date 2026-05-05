# O101

O101 is a Go rewrite of the deprecated **o101** tooling. It focuses on
providing a modern, well-tested patching workflow for managing Wizard101-style
content deployments. The project combines a concurrent patcher, strict manifest
validation, durable state tracking, and an experimental multiplayer realm
emulation server in a single toolkit.

## Features

- **Concurrent patching** – downloads and validates files in parallel with
  bounded concurrency.
- **Deterministic manifests** – JSON manifest format with strict schema
  validation to prevent corrupt deployments.
- **Crash-safe state** – atomic persistence of patch state with automatic
  recovery from partial or corrupted files.
- **CLI first** – the `wizturtle` patcher, `wizserver` realm emulator, and
  `messagesorter` capture analyser deliver streamlined tooling for patching,
  gameplay prototyping, and protocol reverse engineering.
- **WAD + serializer primitives** – production-ready byte buffers, WAD archive
  readers, and serializer helpers compatible with the Open101 reference
  formats.
- **Comprehensive tests** – aggressive unit tests cover manifest parsing,
  state persistence, in-memory realm behaviour, and end-to-end patch execution.

## Installation

```bash
go install github.com/cybellereaper/open101/cmd/wizturtle@latest
go install github.com/cybellereaper/open101/cmd/wizserver@latest
```

## Usage

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

The referenced manifest contains an array of files to install:

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

### Realm Emulator

The `wizserver` command provides a lightweight TCP server that emulates the
login and game sockets exposed by the original C# reference implementation. It
bootstraps a fully in-memory realm with zone assignment and character caching
logic.

```bash
wizserver \
  --game-dir /path/to/game \
  --login-addr 127.0.0.1:12500 \
  --game-addr 127.0.0.1:12501 \
  --max-players 100 \
  --zones 50 \
  --zone-capacity 10
```

New TCP connections receive an informational greeting and are echoed back the
first line of data they send, mirroring the handshake used for integration
tests. The realm package exposes reusable building blocks (`Realm`, `Zone`,
`InGameCharacter`) for embedding in more advanced prototypes.

### Open101 IO & Serialization

The `internal/open101/io` package provides a bit-level `ByteBuffer`, a
production-quality WAD archive parser, and a concurrency-safe resource manager
that mirrors the behaviour of the legacy C# tooling. The accompanying
`internal/open101/serializer` package exposes ergonomic helpers (`ByteString`,
`GID`, and the `BasicBinarySerializer`) for encoding and decoding Wizard101
network payloads.

Aggressive unit tests validate bit packing edge cases, compressed and
uncompressed WAD entries, resource manager caching, and round-trip serialization
of primitive and identifier types.

### Message Capture Sorting

The `messagesorter` command mirrors the legacy MessageSorter utility. It reads a
captured protocol transcript, extracts the service metadata, strips record
blocks, deduplicates protocol messages, and writes an alphabetised listing with
line numbers.

```bash
messagesorter /path/to/MessageSorter/Traffic/ServiceCapture.xml
# wrote 56 messages for service LoginService (42) to ./42_LoginService.txt
```

The output filename follows the `<ServiceID>_<ServiceName>.txt` convention and
defaults to the same directory as the input capture unless `--out` is
specified.

## Development

Run the full test suite:

```bash
go test ./...
```

## License

GNU General Public License v3.0
