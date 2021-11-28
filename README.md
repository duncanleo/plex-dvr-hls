# Plex DVR Emulator (HLS)
This web server emulates a SiliconDust HDHomeRun by its HTTP API for use with Plex's DVR feature. It is designed for use with HLS .m3u8 streams, although any input format accepted by `ffmpeg` should work.

### Features
- Multiple channels
- XMLTV file generation (it just creates a generic 24/7 programme for each available channel)

### Running
1. Download a binary release from GitHub, or clone the repository and compile on your machine (e.g. with `GOOS=linux GOARCH=amd64 go build -o plex-dvr-hls-linux-amd64 cmd/main.go`)
1. Create a `config.json` in the working directory and fill in the necessary. Possible values for `encoder_profile` are `vaapi`, `video_toolbox`, `omx` and `cpu`. A sample `config.example.json` is available on GitHub.
2. Create a `channels.json` and fill in the necessary. A sample `channels.example.json` is available on GitHub.
3. Copy the `templates` folder from this repository into the working directory (alongside the two JSON files)
4. Add the server to the Plex DVR e.g. `http://<ip of machine>:5004`. When prompted for an Electronic Programme Guide, you can either use one if it's available, or use the auto-generated one by entering `http://<ip of machine>:5004/xmltv`

### Development
1. Clone the repo
2. Run `go mod download`
3. To run the server, run `go run cmd/main.go`
