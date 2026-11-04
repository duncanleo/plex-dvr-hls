package config

import (
	"fmt"
	"math"
	"net/url"
	"strings"
)

// InputOptions controls how ffmpeg reads a channel's source. Every field is a
// pointer so that "unset" is distinct from a zero value (for example,
// HLSLiveStartIndex 0 means "start at the oldest segment"). Unset fields are
// omitted from the ffmpeg command, leaving ffmpeg's own defaults in place.
//
// These options exist for live HLS sources with long segments (~10 s). Without
// pacing, ffmpeg's HLS demuxer forwards each segment as soon as it downloads,
// so Plex receives an initial burst followed by multi-second gaps and clients
// buffer. Realtime (-re) paces output at 1x so the initial fetch becomes a
// cushion that absorbs those gaps; RealtimeInitialBurst gives Plex a startup
// buffer before that pacing starts.
type InputOptions struct {
	// Realtime adds -re: read the input at its native rate.
	Realtime *bool `json:"realtime,omitempty"`
	// RealtimeInitialBurst adds -readrate_initial_burst: seconds of input
	// read at full speed before -re pacing starts. ffmpeg then stays this far
	// ahead of real time. It is only emitted when Realtime is on, because
	// ffmpeg ignores it without -re. ffmpeg's default is 0.5.
	RealtimeInitialBurst *float64 `json:"realtimeInitialBurst,omitempty"`
	// HLSLiveStartIndex adds -live_start_index (HLS only). Must be <= 0:
	// negative values count back from the live edge; ffmpeg's default is -3.
	HLSLiveStartIndex *int `json:"hlsLiveStartIndex,omitempty"`
	// HLSSegMaxRetry adds -seg_max_retry (HLS only). ffmpeg's default of 0
	// skips a segment after a single failed download.
	HLSSegMaxRetry *int `json:"hlsSegMaxRetry,omitempty"`
	// HLSMaxReload adds -max_reload (HLS only): consecutive playlist reloads
	// that bring no new segment (including failed reloads) before the stream
	// ends. The default is version dependent: 3 in ffmpeg 6.1, 100 in later
	// releases.
	HLSMaxReload *int `json:"hlsMaxReload,omitempty"`
}

// InputConfig is the config.json form of InputOptions. It uses snake_case keys
// to match the rest of config.json and supplies defaults for every channel,
// including channels loaded from PLAYLIST.
type InputConfig struct {
	Realtime             *bool    `json:"realtime,omitempty"`
	RealtimeInitialBurst *float64 `json:"realtime_initial_burst,omitempty"`
	HLSLiveStartIndex    *int     `json:"hls_live_start_index,omitempty"`
	HLSSegMaxRetry       *int     `json:"hls_seg_max_retry,omitempty"`
	HLSMaxReload         *int     `json:"hls_max_reload,omitempty"`
}

func (c *InputConfig) options() *InputOptions {
	if c == nil {
		return nil
	}
	return &InputOptions{
		Realtime:             c.Realtime,
		RealtimeInitialBurst: c.RealtimeInitialBurst,
		HLSLiveStartIndex:    c.HLSLiveStartIndex,
		HLSSegMaxRetry:       c.HLSSegMaxRetry,
		HLSMaxReload:         c.HLSMaxReload,
	}
}

// Validate rejects values ffmpeg would refuse at startup, or that would break
// live playback, so a bad setting is reported when configuration loads rather
// than when a channel is tuned. ffmpeg stores all three HLS options as 32-bit
// ints, so larger values make ffmpeg fail to start.
func (o *InputOptions) Validate() error {
	if o == nil {
		return nil
	}
	if o.RealtimeInitialBurst != nil {
		// ffmpeg rejects a negative burst at startup.
		if v := *o.RealtimeInitialBurst; v < 0 || math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("realtime initial burst must be a non-negative number of seconds, got %v", v)
		}
	}
	if o.HLSLiveStartIndex != nil {
		v := *o.HLSLiveStartIndex
		// Positive values count from the oldest segment, which on a live
		// playlist is the next to expire.
		if v > 0 {
			return fmt.Errorf("hls live start index must be <= 0 (counts back from the live edge), got %d", v)
		}
		if int64(v) < math.MinInt32 {
			return fmt.Errorf("hls live start index must be >= %d, got %d", math.MinInt32, v)
		}
	}
	if err := checkCount("hls seg max retry", o.HLSSegMaxRetry); err != nil {
		return err
	}
	return checkCount("hls max reload", o.HLSMaxReload)
}

// checkCount validates a retry-style count against ffmpeg's 0..INT32_MAX range.
func checkCount(name string, p *int) error {
	if p == nil {
		return nil
	}
	if v := int64(*p); v < 0 || v > math.MaxInt32 {
		return fmt.Errorf("%s must be between 0 and %d, got %d", name, math.MaxInt32, *p)
	}
	return nil
}

// HasHLSOptions reports whether any HLS-only option is set.
func (o *InputOptions) HasHLSOptions() bool {
	return o != nil && (o.HLSLiveStartIndex != nil || o.HLSSegMaxRetry != nil || o.HLSMaxReload != nil)
}

// ResolveInputOptions merges per-channel options over the config.json
// defaults field by field. A field set on the channel wins; otherwise the
// global value is used; otherwise the field stays unset. The result is never
// nil.
func ResolveInputOptions(cfg Config, channel Channel) *InputOptions {
	resolved := &InputOptions{}
	if global := cfg.Input.options(); global != nil {
		*resolved = *global
	}
	if channel.Input == nil {
		return resolved
	}
	if channel.Input.Realtime != nil {
		resolved.Realtime = channel.Input.Realtime
	}
	if channel.Input.RealtimeInitialBurst != nil {
		resolved.RealtimeInitialBurst = channel.Input.RealtimeInitialBurst
	}
	if channel.Input.HLSLiveStartIndex != nil {
		resolved.HLSLiveStartIndex = channel.Input.HLSLiveStartIndex
	}
	if channel.Input.HLSSegMaxRetry != nil {
		resolved.HLSSegMaxRetry = channel.Input.HLSSegMaxRetry
	}
	if channel.Input.HLSMaxReload != nil {
		resolved.HLSMaxReload = channel.Input.HLSMaxReload
	}
	return resolved
}

// IsHLSURL reports whether the source URL points at an HLS playlist. HLS-only
// ffmpeg options must not be passed to other inputs: ffmpeg rejects unknown
// demuxer options and the channel would fail to tune.
func IsHLSURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
}
