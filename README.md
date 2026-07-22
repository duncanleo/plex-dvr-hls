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
4. Add the server to the Plex DVR e.g. `http://<ip of machine>:5004`.
   - When prompted for an Electronic Programme Guide, you can either use one if it's available, or use the auto-generated one by entering `http://<ip of machine>:5004/xmltv`

### Environment Variables

| Variable | Description |
|---|---|
| `PORT` | Port to listen on (default: `5004`) |
| `PLAYLIST` | URL or path to an M3U playlist. When set, channels are loaded from the playlist instead of `channels.json`. |
| `UA` | User-Agent sent when fetching the M3U playlist (default: Chrome UA string) |

### Development
1. Clone the repo
2. Run `go mod download`
3. To run the server, run `go run cmd/main.go`

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
