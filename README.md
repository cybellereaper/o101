# OpenArcadia

OpenArcadia is a unified infrastructure toolkit for Wizard-style online worlds. It combines content deployment, protocol utilities, WAD archive support, message capture analysis, serialization primitives, and experimental multiplayer realm services into one platform.

## Features

- **Content deployment** — concurrent patching with strict manifests, validation, and crash-safe state tracking.
- **Archive support** — WAD parsing, CRC validation, compression handling, and resource management.
- **Protocol tooling** — capture analysis, serializers, and reusable networking primitives.
- **Realm simulation** — experimental multiplayer world services with zones, characters, and session handling.
- **Developer-first tooling** — one ecosystem with shared libraries and consistent command naming.

## Project layout

```
openarcadia
├── crates/
│   ├── archive       # WAD and resource handling
│   ├── codec         # Serialization primitives
│   ├── deployment    # Patch workflows
│   ├── protocol      # Message and network utilities
│   └── realm         # Multiplayer simulation
│
└── tools/
    ├── arcadia       # Main CLI
    ├── arc-patcher   # Deployment tooling
    ├── arc-inspect   # Protocol analysis
    └── arc-realm     # Realm server
```

## Installation

```bash
cargo install --git https://github.com/cybellereaper/o101 --bin arcadia
```

## Development

```bash
cargo fmt --all -- --check
cargo clippy --all-targets --all-features -- -D warnings
cargo test --all-targets
cargo build --release --bins
```

## License

GNU General Public License v3.0
