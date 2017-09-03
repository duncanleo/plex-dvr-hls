# Plex DVR Emulator (HLS)
This web server emulates a SiliconDust HDHomeRun by its HTTP API for use with Plex's DVR feature. It is designed for use with HLS .m3u8 streams, although any input format accepted by `ffmpeg` should work.

## Running
1. `yarn install`
2. `yarn start`
3. Add the server to the Plex DVR

## Features
- Multiple channels
- XMLTV file generation (it just creates a generic 24/7 programme for each available channel)
