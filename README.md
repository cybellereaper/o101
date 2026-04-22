# O101 (Crystal Edition)

O101 is a Crystal rewrite of the legacy Go implementation. The project keeps the
same toolkit goals while improving separation of concerns and testability:

- `wizturtle` for patching content deployments
- `wizserver` for lightweight realm emulation
- `messagesorter` for protocol transcript extraction
- reusable Open101 IO/serializer primitives

## Build

```bash
shards install
crystal build src/bin/wizturtle.cr -o bin/wizturtle
crystal build src/bin/wizserver.cr -o bin/wizserver
crystal build src/bin/messagesorter.cr -o bin/messagesorter
```

## Test

```bash
crystal spec
```

## Commands

### wizturtle

```bash
bin/wizturtle \
  --patch-info https://example.com/patch-info.json \
  --install-dir /path/to/install \
  --state-file /path/to/install/.wizturtle/state.json
```

### wizserver

```bash
bin/wizserver --login-addr 127.0.0.1:12500 --game-addr 127.0.0.1:12501
```

### messagesorter

```bash
bin/messagesorter ./ServiceCapture.xml --out ./out
```

## License

BSD 3-Clause
