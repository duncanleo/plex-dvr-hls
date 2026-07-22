# Operations

## Runtime inputs

The process expects `config.json` in its working directory. It loads channels
from `channels.json` unless `PLAYLIST` is set. Copy
`config.example.json` and `channels.example.json` before running locally; do
not commit populated runtime files.

| Input | Purpose | Notes |
| --- | --- | --- |
| `config.json` | DVR name, encoder profile, tuner count, optional device ID | Required. |
| `channels.json` | Channel sources and per-channel options | Watched and hot-reloaded when `PLAYLIST` is unset. |
| `PLAYLIST` | M3U URL or file path | Takes precedence over `channels.json`. |
| `PORT` | HTTP listen port | Defaults to `5004`. |
| `UA` | Playlist-fetch User-Agent | Used only for playlist loading. |

## Local development

```sh
cp config.example.json config.json
cp channels.example.json channels.json
go run cmd/main.go
```

The service invokes `ffmpeg` only when a stream endpoint is requested. Install
an `ffmpeg` build compatible with the selected encoder profile before manual
stream verification.

## Docker

`docker-compose.yml` bind-mounts configuration files into `/app`, which is the
container working directory. Use read-only mounts for inputs. If a stable DVR
device ID matters, set `device_id` explicitly in `config.json`; otherwise the
app attempts to persist `.device_id` in the working directory.

## Failure behaviour

- Invalid startup configuration or an unavailable initial channel source stops
  the process because no usable state exists.
- A failed channel reload logs the error and retains the previous list.
- A missing or invalid stream channel returns a client error without starting
  `ffmpeg`.
- A short-lived `ffmpeg` process is retried a bounded number of times. After
  the threshold, an initial no-output stream returns a gateway error.
- SIGINT/SIGTERM cancels active stream processes before HTTP server shutdown.

When changing any of these behaviours, update the related tests and
[ARCHITECTURE.md](../ARCHITECTURE.md).
