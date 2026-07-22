package routes

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/duncanleo/plex-dvr-hls/config"
	"github.com/gin-gonic/gin"
)

// execCommandContext is the exec.CommandContext constructor. Tests override this
// to inject a fake ffmpeg process without modifying PATH.
var execCommandContext func(ctx context.Context, name string, arg ...string) *exec.Cmd = exec.CommandContext

// streamMaxFailures, streamRetryDelay, and minHealthyDuration control the
// restart loop (fixed delay, no exponential backoff). Tests override
// streamRetryDelay and minHealthyDuration to avoid multi-second delays.
var (
	streamMaxFailures  = 3
	streamRetryDelay   = 2 * time.Second
	minHealthyDuration = 10 * time.Second
)

// flushWriter wraps gin.ResponseWriter to flush after each Write, ensuring
// MPEG-TS chunks reach the client without waiting for Go's write buffer to fill.
type flushWriter struct {
	gin.ResponseWriter
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.ResponseWriter.Write(p)
	if err == nil {
		fw.ResponseWriter.Flush()
	}
	return n, err
}

func buildFFmpegArgs(channel config.Channel, transcode string) []string {
	var args []string

	if channel.ProxyConfig != nil {
		args = append(args,
			"-http_proxy",
			fmt.Sprintf("http://%s:%s@%s", channel.ProxyConfig.Username, channel.ProxyConfig.Password, channel.ProxyConfig.Host),
		)
	}

	// ffmpeg -headers takes a single dict; multiple -headers flags overwrite each other.
	// Build one combined value so both UserAgent and Referer reach the origin.
	var headers string
	if channel.UserAgent != nil {
		headers += fmt.Sprintf("User-Agent: %s\r\n", *channel.UserAgent)
	}
	if channel.Referer != nil {
		headers += fmt.Sprintf("Referer: %s\r\n", *channel.Referer)
	}
	if headers != "" {
		args = append(args, "-headers", headers)
	}

	switch config.Cfg.GetEncoderProfile() {
	case config.EncoderProfileVAAPI:
		args = append(args,
			"-vaapi_device", "/dev/dri/renderD128",
			"-hwaccel", "vaapi",
			"-hwaccel_output_format", "vaapi",
		)
	case config.EncoderProfileVideoToolbox:
		args = append(args, "-hwaccel", "videotoolbox")
	case config.EncoderProfileNVENC:
		args = append(args,
			"-hwaccel", "cuda",
			"-hwaccel_output_format", "cuda",
		)
	}

	// -reconnect* flags are HTTP(S)-only; passing them to RTSP or other
	// protocols causes ffmpeg to error on startup.
	// -reconnect_at_eof/-reconnect_streamed are opt-in via channel.Reconnect
	// (see Channel.Reconnect doc for why they are off by default).
	if strings.HasPrefix(channel.URL, "http://") || strings.HasPrefix(channel.URL, "https://") {
		args = append(args, "-reconnect", "1", "-reconnect_delay_max", "5")
		if channel.Reconnect {
			args = append(args, "-reconnect_at_eof", "1", "-reconnect_streamed", "1")
		}
	}

	args = append(args, "-i", channel.URL)

	if channel.DisableTranscode {
		args = append(args, "-c:v", "copy")
	} else {
		switch config.Cfg.GetEncoderProfile() {
		case config.EncoderProfileVideoToolbox:
			args = append(args, "-c:v", "h264_videotoolbox")
		case config.EncoderProfileVAAPI:
			args = append(args, "-c:v", "h264_vaapi", "-vf", "scale_vaapi=format=nv12,hwupload")
		case config.EncoderProfileOMX:
			args = append(args, "-c:v", "h264_omx")
		case config.EncoderProfileNVENC:
			args = append(args, "-c:v", "h264_nvenc", "-preset", "p3")
		default:
			args = append(args, "-c:v", "libx264", "-preset", "superfast")
		}
	}

	if channel.DisableAudioTranscode {
		args = append(args, "-c:a", "copy")
	} else {
		args = append(args, "-b:a", "256k")
	}

	args = append(args,
		"-copyinkf",
		"-metadata", "service_provider=AMAZING",
		"-metadata", fmt.Sprintf("service_name=%s", strings.ReplaceAll(channel.Name, " ", "")),
		"-tune", "zerolatency",
		"-mbd", "rd",
		"-flags", "+ilme+ildct",
		"-fflags", "+genpts",
	)

	switch transcode {
	case "mobile", "internet720":
		args = append(args, "-s", "1280x720", "-r", "30")
	}

	args = append(args, "-f", "mpegts", "pipe:1")

	return args
}

// StreamWithContext returns a handler that streams the channel's video.
// serverCtx is cancelled on SIGTERM so in-flight ffmpeg processes are killed
// cleanly on shutdown rather than orphaned.
func StreamWithContext(serverCtx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		channelID, err := strconv.Atoi(c.Param("channelID"))
		if err != nil {
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}

		channel, ok := config.GetChannel(channelID - 1)
		if !ok {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		transcode := c.Query("transcode")
		log.Printf("[STREAM] Starting '%s'\n", channel.Name)
		c.Header("Content-Type", "video/mp2t")

		// ctx is cancelled when either the client disconnects or the server shuts down.
		ctx, cancel := context.WithCancel(serverCtx)
		defer cancel()
		go func() {
			<-c.Request.Context().Done()
			cancel()
		}()

		args := buildFFmpegArgs(channel, transcode)
		quickRestarts := 0
		var totalWritten int64

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			startTime := time.Now()
			ffmpegProcess := execCommandContext(ctx, "ffmpeg", args...)

			outPipe, pipeErr := ffmpegProcess.StdoutPipe()
			if pipeErr != nil {
				log.Printf("[STREAM] pipe error for '%s': %v\n", channel.Name, pipeErr)
				if totalWritten == 0 {
					c.AbortWithStatus(http.StatusInternalServerError)
				}
				return
			}
			ffmpegProcess.Stderr = os.Stdout

			if startErr := ffmpegProcess.Start(); startErr != nil {
				log.Printf("[STREAM] ffmpeg failed to start for '%s': %v\n", channel.Name, startErr)
				if totalWritten == 0 {
					c.AbortWithStatus(http.StatusInternalServerError)
				}
				return
			}

			n, copyErr := io.Copy(flushWriter{c.Writer}, outPipe)
			totalWritten += n

			if waitErr := ffmpegProcess.Wait(); waitErr != nil {
				log.Printf("[STREAM] ffmpeg exited for '%s': %v\n", channel.Name, waitErr)
			}

			// Check context cancellation first: a killed ffmpeg gives copyErr==nil
			// (pipe closes cleanly), so checking copyErr alone misses this case.
			if ctx.Err() != nil {
				log.Printf("[STREAM] stream for '%s' stopped (%v) after %d bytes\n", channel.Name, ctx.Err(), totalWritten)
				return
			}

			if copyErr != nil {
				log.Printf("[STREAM] copy error for '%s' after %d bytes: %v\n", channel.Name, totalWritten, copyErr)
				return
			}

			// Count any short-lived exit (0-byte or bytes-then-die) as a quick
			// restart. Only a run that lasted minHealthyDuration resets the counter,
			// preventing alternating 0-byte / trivial-byte patterns from bypassing
			// the cap.
			if time.Since(startTime) < minHealthyDuration {
				quickRestarts++
				if quickRestarts >= streamMaxFailures {
					log.Printf("[STREAM] ffmpeg exited quickly %d times for '%s', giving up\n",
						streamMaxFailures, channel.Name)
					if totalWritten == 0 {
						c.AbortWithStatus(http.StatusBadGateway)
					}
					return
				}
				log.Printf("[STREAM] ffmpeg exited quickly for '%s' (restart %d/%d), retrying in %v\n",
					channel.Name, quickRestarts, streamMaxFailures, streamRetryDelay)
			} else {
				quickRestarts = 0
				log.Printf("[STREAM] Restarting ffmpeg for '%s'\n", channel.Name)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(streamRetryDelay):
			}
		}
	}
}
