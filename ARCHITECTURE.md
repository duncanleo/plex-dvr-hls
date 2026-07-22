# Architecture

## Purpose

`plex-dvr-hls` presents a small subset of the SiliconDust HDHomeRun HTTP API
to Plex. It exposes discovery, lineup, status, XMLTV, and streaming endpoints.
Each stream starts an `ffmpeg` process that reads the configured source and
writes MPEG-TS to the requesting client.

## Runtime flow

```text
config.json / channels.json / PLAYLIST
                  |
                  v
              config package
                  |
                  v
cmd/main.go -> Gin router -> routes handlers -> ffmpeg -> Plex client
                  |
                  +-> shutdown context -> active stream processes
```

`config` initializes configuration and channels at process startup. When using
`channels.json`, it watches the file and swaps the in-memory channel snapshot
only after a successful reload. `cmd/main.go` owns server lifecycle and passes
a cancellation context into the streaming route.

## Package boundaries

| Package | Owns | May depend on |
| --- | --- | --- |
| `cmd` | Router registration, listening, signals, graceful shutdown | `config`, `routes` |
| `config` | Runtime configuration, channel model, file/playlist loading, watcher, synchronization | Standard library and loader dependencies |
| `routes` | HTTP representation, XMLTV rendering, stream process lifecycle | `config`, Gin, standard library |

Dependency direction is one way: `cmd → routes → config`. Do not introduce a
dependency from `config` to HTTP concerns or from `routes` to `cmd`.

## Important invariants

- Configuration is read from the working directory. Docker sets this to
  `/app`; local execution uses the repository root unless the caller changes
  directory.
- `config.Cfg.DeviceID` is always populated after successful configuration
  loading. Its priority is `config.json`, then `.device_id`, then a generated
  value.
- Channel reads and replacements are synchronized. Callers receive a channel
  copy or a copied slice.
- A zero or malformed replacement channel list is rejected and the previous
  valid list stays active.
- Stream requests use one-based `channelID` values. The corresponding list
  position can change after reload.
- `ffmpeg` is tied to both request cancellation and process shutdown. A stream
  must not leave a child process running after either occurs.

## Change guide

| Change | Update and verify |
| --- | --- |
| New endpoint or HTTP response | Route tests, this document, README user-facing endpoint guidance if applicable. |
| New config/channel field | `config` tests, examples, README, and `docs/OPERATIONS.md`. |
| Reload/concurrency change | `config` tests and `make test-race`. |
| Stream command/retry change | `routes/stream_test.go`, `docs/OPERATIONS.md`, and failure behaviour. |
| Package responsibility changes | This document and `AGENTS.md`. |
