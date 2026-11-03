# Plex DVR Emulator (HLS)
This web server emulates a SiliconDust HDHomeRun by its HTTP API for use with Plex's DVR feature. It is designed for use with HLS .m3u8 streams, although any input format accepted by `ffmpeg` should work.

### Features
- Multiple channels
- XMLTV file generation (generic 24/7 programme per channel)
- Per-channel video/audio passthrough (skip re-encoding for better quality)
- Per-channel proxy, custom User-Agent, and Referer support
- Automatic ffmpeg restart on upstream stream drops (2s retry delay; gives up after 3 consecutive exits that each lasted under 10s)
- Hot-reload of `channels.json` without restart

### Running
##### Docker
A prebuilt [Docker image](https://github.com/duncanleo/plex-dvr-hls/pkgs/container/plex-dvr-hls) is available for use. 

###### Supported Architectures
- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

###### Docker Compose
Please refer to the [sample Docker Compose file](./docker-compose.yml) for a more seamless setup.

Please note that the following files need to be present in the same directory (see examples in the repository).
- `config.json`
- `channels.json`

```yaml
services:
  plex-dvr-hls:
    image: ghcr.io/duncanleo/plex-dvr-hls:latest
    volumes:
      - type: bind
        source: './config.json'
        target: '/app/config.json'
        read_only: true
      - type: bind
        source: './channels.json'
        target: '/app/channels.json'
        read_only: true
    ports:
      - '5004:5004'
```

##### Binary
1. Download a binary release from GitHub, or clone the repository and compile on your machine (e.g. with `GOOS=linux GOARCH=amd64 go build -o plex-dvr-hls-linux-amd64 cmd/main.go`)
2. Create a `config.json` in the working directory and fill in the necessary.
   - Possible values for `encoder_profile` are `vaapi`, `video_toolbox`, `omx`, `nvenc` and `cpu`. A sample `config.example.json` is available on GitHub.
     - `nvenc` requires an NVIDIA GPU and ffmpeg with NVENC support. For Docker, see [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html).
   - **Optional:** Set `device_id` to a specific value (e.g. `"30480554"`) to maintain a stable device ID. If omitted, a random ID will be generated on first run and automatically saved to `.device_id` for persistence across restarts. Priority order: `config.json` > `.device_id` file > auto-generate new.
   - **Optional:** Set `input` to apply ffmpeg input options to every channel (including channels loaded from `PLAYLIST`). See [Input options](#input-options).
3. Create a `channels.json` and fill in the necessary.
   - A sample `channels.example.json` is available on GitHub.
   - Each channel supports the following fields:
     | Field | Type | Description |
     |---|---|---|
     | `name` | string | Display name shown in Plex |
     | `url` | string | Stream URL (M3U8, RTSP, or any ffmpeg-supported input) |
     | `disableTranscode` | bool | Pass video through unchanged (`-c:v copy`) instead of re-encoding |
     | `disableAudioTranscode` | bool | Pass audio through unchanged (`-c:a copy`) instead of re-encoding at 256k |
     | `reconnect` | bool | Enable `-reconnect_at_eof` and `-reconnect_streamed` for this channel. **Off by default** — HLS sources must leave this off (causes infinite reconnect loop on providers with rotating 302 redirects). Enable only for direct MPEG-TS or other non-HLS HTTP sources (e.g. `http://host/stream.ts`). |
     | `userAgent` | string | Custom `User-Agent` header sent to the stream source |
     | `referer` | string | Custom `Referer` header sent to the stream source |
     | `icon` | string | URL to channel logo (used in XMLTV EPG) |
     | `proxy.host` | string | HTTP proxy host and port (e.g. `proxy.example.com:3128`) |
     | `proxy.username` | string | Proxy authentication username |
     | `proxy.password` | string | Proxy authentication password |
     | `input` | object | Per-channel ffmpeg input options; each field overrides the `config.json` `input` value. See [Input options](#input-options). |
4. Add the server to the Plex DVR e.g. `http://<ip of machine>:5004`.
   - When prompted for an Electronic Programme Guide, you can either use one if it's available, or use the auto-generated one by entering `http://<ip of machine>:5004/xmltv`

### Input options

Optional ffmpeg input options. **All are off by default** — an unset option is left out of the ffmpeg command, so existing setups are unchanged.

| `config.json` (`input`) | `channels.json` (`input`) | ffmpeg flag | Applies to | Purpose |
|---|---|---|---|---|
| `realtime` | `realtime` | `-re` | all inputs | Read the source at its native rate instead of as fast as it downloads |
| `hls_live_start_index` | `hlsLiveStartIndex` | `-live_start_index` | `.m3u8` only | Segment to start live streams at; must be `<= 0` (negative counts back from the live edge; ffmpeg default `-3`) |
| `hls_seg_max_retry` | `hlsSegMaxRetry` | `-seg_max_retry` | `.m3u8` only | Retries for a failed segment download (ffmpeg default `0` skips the segment) |
| `hls_max_reload` | `hlsMaxReload` | `-max_reload` | `.m3u8` only | Consecutive playlist reloads with no new segment (including failed reloads) before the stream ends. Default is version dependent: `3` in ffmpeg 6.1 (the Docker image), `100` in later releases |

Precedence per option: channel value → `config.json` value → unset. `hls_live_start_index` must be `<= 0`, and retry counts between `0` and `2147483647` (ffmpeg stores all three as 32-bit ints); invalid values fail startup, or keep the previous channels on a hot reload.

**Known limitation: HLS detection is by URL extension.** HLS-only options are applied only when the URL path ends in `.m3u8` (case-insensitive; query strings are ignored), because ffmpeg refuses to start when given them on other inputs. Sources that serve HLS from an extensionless URL (for example `http://host/live/user/pass/123`), or that redirect to an `.m3u8`, do not get the HLS-only options; `realtime` still applies. A log line is written when a channel's own `input` sets HLS-only options that are skipped.

**Buffering on live HLS sources with long segments.** Without `-re`, ffmpeg forwards each segment as soon as it downloads. With ~10 s segments, Plex receives a ~30 s burst at tune-in followed by multi-second gaps, and clients (especially remote ones) buffer even though VLC plays the same URL fine. Enabling `realtime` paces output at 1×, so the initial fetch becomes a cushion that absorbs the gaps:

```json
{
  "name": "Amazing Tuner",
  "encoder_profile": "cpu",
  "input": {
    "realtime": true,
    "hls_seg_max_retry": 3,
    "hls_max_reload": 100
  }
}
```

The trade-off is extra delay behind live (roughly `-live_start_index` × segment length). If buffering persists, raise the cushion for that channel with `"input": { "hlsLiveStartIndex": -5 }`. `hls_max_reload: 100` matches the newer ffmpeg default and raises 6.1's limit of `3`, which can end a stream after roughly 15 s without a new segment.

`realtime` stops ffmpeg reading ahead of real time; it does not stop it falling behind after a slow segment download. ffmpeg 6.1 then reads at full speed until it catches up (a short burst). Newer ffmpeg catches up at only 1.05× by default (`-readrate_catchup`, not exposed here), so repeated stalls can push it behind the provider's playlist window, at which point expired segments are skipped. Keep the live start index well inside the playlist window (e.g. `-3` to `-5` for a 6-segment playlist).

### Environment Variables

| Variable | Description |
|---|---|
| `PORT` | Port to listen on (default: `5004`) |
| `PLAYLIST` | URL or path to an M3U playlist. When set, channels are loaded from the playlist instead of `channels.json`. |
| `UA` | User-Agent sent when fetching the M3U playlist (default: Chrome UA string) |

### Development
1. Clone the repo
2. Run `go mod download`
3. Copy `config.example.json` and `channels.example.json` to `config.json` and `channels.json`, then configure them.
4. To run the server, run `go run cmd/main.go`

### Testing
Run the test suite:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests with race detection:
```bash
go test -race ./...
```

### Contributor checks

The repository includes a small, dependency-free engineering harness for
repeatable agent and human contributions. Before opening a change, run:

```bash
make check
```

For changes to concurrent state, file watching, streaming, or shutdown, also
run:

```bash
make test-race
```

See [AGENTS.md](./AGENTS.md) for the concise contributor entry point and
[docs/README.md](./docs/README.md) for project knowledge and execution plans.
