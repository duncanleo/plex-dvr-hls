# Plex DVR Emulator (HLS)
This web server emulates a SiliconDust HDHomeRun by its HTTP API for use with Plex's DVR feature. It is designed for use with HLS .m3u8 streams, although any input format accepted by `ffmpeg` should work.

### Features
- Multiple channels
- XMLTV file generation (it just creates a generic 24/7 programme for each available channel)

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
   - **Optional:** Set `device_id` to a specific value (e.g. `"30480554"`) to maintain a constant device ID across restarts. If omitted, a random ID will be generated once at startup and logged to the console.
3. Create a `channels.json` and fill in the necessary.
   - A sample `channels.example.json` is available on GitHub.
4. Copy the `templates` folder from this repository into the working directory (alongside the two JSON files)
5. Add the server to the Plex DVR e.g. `http://<ip of machine>:5004`.
   - When prompted for an Electronic Programme Guide, you can either use one if it's available, or use the auto-generated one by entering `http://<ip of machine>:5004/xmltv`

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
