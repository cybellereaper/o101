# O101 (Crystal Edition)

O101 is now a Crystal-native toolkit for Wizard101-style patching and protocol workflows.

## Components

- `wizturtle`: patch runner with manifest validation and state persistence.
- `messagesorter`: extracts and deduplicates protocol tags from capture files.
- `wizserver`: lightweight TCP realm prototype for login handshake simulation.
- Core libraries under `src/open101` for manifest parsing, patching, state management,
  WizServer realm logic, WAD primitives, and serializer helpers.

## Build

```bash
shards build
```

## Test

```bash
crystal spec
```
