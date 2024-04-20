# Plex DVR Emulator (HLS)
This web server emulates a SiliconDust HDHomeRun by its HTTP API for use with Plex's DVR feature. It is designed for use with HLS .m3u8 streams, although any input format accepted by `ffmpeg` should work.

### Features
- Multiple channels
- XMLTV file generation (it just creates a generic 24/7 programme for each available channel)

### Running
##### Docker
A Docker container is available for use. Do take note of the need for `config.json` and `channels.json`.

```yaml
services:
    plex-dvr-hls:
        image: ghcr.io/duncanleo/plex-dvr-hls:latest
        volumes:
            - './config.json:/app/config.json:ro'
            - './channels.json:/app/channels.json:ro'
            - './templates:/app/templates:ro'
        ports:
            - '5004:5004'
```

##### Binary
1. Download a binary release from GitHub, or clone the repository and compile on your machine (e.g. with `GOOS=linux GOARCH=amd64 go build -o plex-dvr-hls-linux-amd64 cmd/main.go`)
2. Create a `config.json` in the working directory and fill in the necessary.
   - Possible values for `encoder_profile` are `vaapi`, `video_toolbox`, `omx` and `cpu`. A sample `config.example.json` is available on GitHub.
3. Create a `channels.json` and fill in the necessary.
   - A sample `channels.example.json` is available on GitHub.
4. Copy the `templates` folder from this repository into the working directory (alongside the two JSON files)
5. Add the server to the Plex DVR e.g. `http://<ip of machine>:5004`.
   - When prompted for an Electronic Programme Guide, you can either use one if it's available, or use the auto-generated one by entering `http://<ip of machine>:5004/xmltv`

### Development
1. Clone the repo
2. Run `go mod download`
3. To run the server, run `go run cmd/main.go`
