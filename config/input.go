package config

import (
	"fmt"
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
// cushion that absorbs those gaps.
type InputOptions struct {
	// Realtime adds -re: read the input at its native rate.
	Realtime *bool `json:"realtime,omitempty"`
	// HLSLiveStartIndex adds -live_start_index (HLS only). Negative values
	// count back from the live edge; ffmpeg's default is -3.
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
	Realtime          *bool `json:"realtime,omitempty"`
	HLSLiveStartIndex *int  `json:"hls_live_start_index,omitempty"`
	HLSSegMaxRetry    *int  `json:"hls_seg_max_retry,omitempty"`
	HLSMaxReload      *int  `json:"hls_max_reload,omitempty"`
}

func (c *InputConfig) options() *InputOptions {
	if c == nil {
		return nil
	}
	return &InputOptions{
		Realtime:          c.Realtime,
		HLSLiveStartIndex: c.HLSLiveStartIndex,
		HLSSegMaxRetry:    c.HLSSegMaxRetry,
		HLSMaxReload:      c.HLSMaxReload,
	}
}

// Validate rejects values ffmpeg would refuse at startup, so a bad setting is
// reported when configuration loads rather than when a channel is tuned.
func (o *InputOptions) Validate() error {
	if o == nil {
		return nil
	}
	if o.HLSSegMaxRetry != nil && *o.HLSSegMaxRetry < 0 {
		return fmt.Errorf("hls seg max retry must be >= 0, got %d", *o.HLSSegMaxRetry)
	}
	if o.HLSMaxReload != nil && *o.HLSMaxReload < 0 {
		return fmt.Errorf("hls max reload must be >= 0, got %d", *o.HLSMaxReload)
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
