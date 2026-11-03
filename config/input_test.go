package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func intPtr(i int) *int    { return &i }
func boolPtr(b bool) *bool { return &b }

// TestResolveInputOptionsUnset verifies that with nothing configured every
// option stays unset, so buildFFmpegArgs leaves the command unchanged.
func TestResolveInputOptionsUnset(t *testing.T) {
	got := ResolveInputOptions(Config{}, Channel{})
	if got.Realtime != nil || got.HLSLiveStartIndex != nil || got.HLSSegMaxRetry != nil || got.HLSMaxReload != nil {
		t.Errorf("expected all options unset, got %+v", got)
	}
}

// TestResolveInputOptionsPrecedence verifies the channel → global → unset
// precedence is applied per field rather than per struct.
func TestResolveInputOptionsPrecedence(t *testing.T) {
	cfg := Config{Input: &InputConfig{
		Realtime:          boolPtr(true),
		HLSLiveStartIndex: intPtr(-3),
		HLSSegMaxRetry:    intPtr(3),
	}}
	channel := Channel{Input: &InputOptions{
		Realtime:          boolPtr(false),
		HLSLiveStartIndex: intPtr(-5),
	}}

	got := ResolveInputOptions(cfg, channel)

	if got.Realtime == nil || *got.Realtime {
		t.Errorf("Realtime: channel false should override global true, got %v", got.Realtime)
	}
	if got.HLSLiveStartIndex == nil || *got.HLSLiveStartIndex != -5 {
		t.Errorf("HLSLiveStartIndex: expected channel -5, got %v", got.HLSLiveStartIndex)
	}
	if got.HLSSegMaxRetry == nil || *got.HLSSegMaxRetry != 3 {
		t.Errorf("HLSSegMaxRetry: expected global 3, got %v", got.HLSSegMaxRetry)
	}
	if got.HLSMaxReload != nil {
		t.Errorf("HLSMaxReload: expected unset, got %d", *got.HLSMaxReload)
	}
}

// TestResolveInputOptionsChannelOnly verifies channel options apply when
// config.json has no input section.
func TestResolveInputOptionsChannelOnly(t *testing.T) {
	got := ResolveInputOptions(Config{}, Channel{Input: &InputOptions{HLSMaxReload: intPtr(100)}})
	if got.HLSMaxReload == nil || *got.HLSMaxReload != 100 {
		t.Errorf("expected channel-only HLSMaxReload 100, got %v", got.HLSMaxReload)
	}
}

func TestInputOptionsValidate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *InputOptions
		wantErr bool
	}{
		{"nil", nil, false},
		{"empty", &InputOptions{}, false},
		{"valid", &InputOptions{Realtime: boolPtr(true), HLSLiveStartIndex: intPtr(-5), HLSSegMaxRetry: intPtr(3), HLSMaxReload: intPtr(100)}, false},
		{"zero retries", &InputOptions{HLSSegMaxRetry: intPtr(0), HLSMaxReload: intPtr(0)}, false},
		{"positive live start index", &InputOptions{HLSLiveStartIndex: intPtr(2)}, false},
		{"negative seg retry", &InputOptions{HLSSegMaxRetry: intPtr(-1)}, true},
		{"negative max reload", &InputOptions{HLSMaxReload: intPtr(-1)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsHLSURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"http://example.com/live/index.m3u8", true},
		{"https://example.com/live/INDEX.M3U8", true},
		{"https://example.com/live/index.m3u8?token=abc", true},
		{"http://example.com/stream.ts", false},
		{"http://example.com/stream?format=m3u8", false},
		{"rtsp://example.com/stream", false},
		{"://bad url", false},
	}
	for _, tt := range tests {
		if got := IsHLSURL(tt.url); got != tt.want {
			t.Errorf("IsHLSURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

// TestInputJSONTags verifies config.json uses snake_case keys and
// channels.json uses camelCase keys. A mistyped tag would silently leave the
// option unset.
func TestInputJSONTags(t *testing.T) {
	var cfg Config
	cfgJSON := `{"input":{"realtime":true,"hls_live_start_index":-5,"hls_seg_max_retry":3,"hls_max_reload":100}}`
	if err := json.Unmarshal([]byte(cfgJSON), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Input == nil || cfg.Input.Realtime == nil || !*cfg.Input.Realtime ||
		cfg.Input.HLSLiveStartIndex == nil || *cfg.Input.HLSLiveStartIndex != -5 ||
		cfg.Input.HLSSegMaxRetry == nil || *cfg.Input.HLSSegMaxRetry != 3 ||
		cfg.Input.HLSMaxReload == nil || *cfg.Input.HLSMaxReload != 100 {
		t.Errorf("config.json input not fully decoded: %+v", cfg.Input)
	}

	var ch Channel
	chJSON := `{"name":"c","url":"http://x/a.m3u8","input":{"realtime":true,"hlsLiveStartIndex":-5,"hlsSegMaxRetry":3,"hlsMaxReload":100}}`
	if err := json.Unmarshal([]byte(chJSON), &ch); err != nil {
		t.Fatal(err)
	}
	if ch.Input == nil || ch.Input.Realtime == nil || !*ch.Input.Realtime ||
		ch.Input.HLSLiveStartIndex == nil || *ch.Input.HLSLiveStartIndex != -5 ||
		ch.Input.HLSSegMaxRetry == nil || *ch.Input.HLSSegMaxRetry != 3 ||
		ch.Input.HLSMaxReload == nil || *ch.Input.HLSMaxReload != 100 {
		t.Errorf("channels.json input not fully decoded: %+v", ch.Input)
	}
}

// TestLoadChannelsFromFileInvalidInputKeepsPrevious verifies that an invalid
// input option fails the reload and retains the previous channel list.
func TestLoadChannelsFromFileInvalidInputKeepsPrevious(t *testing.T) {
	saved := GetChannels()
	t.Cleanup(func() { SetChannels(saved) })
	SetChannels([]Channel{{Name: "before", URL: "http://a.com"}})

	path := filepath.Join(t.TempDir(), "channels.json")
	content := `[{"name":"bad","url":"http://a.com/x.m3u8","input":{"hlsSegMaxRetry":-1}}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	err := loadChannelsFromFile(path)
	if err == nil || !strings.Contains(err.Error(), `"bad"`) {
		t.Fatalf("expected error naming the channel, got %v", err)
	}
	ch, ok := GetChannel(0)
	if !ok || ch.Name != "before" {
		t.Errorf("channels modified after invalid reload: got %+v", ch)
	}
}

// TestLoadChannelsFromFileValidInput verifies a valid per-channel input block
// loads through the real reload path.
func TestLoadChannelsFromFileValidInput(t *testing.T) {
	saved := GetChannels()
	t.Cleanup(func() { SetChannels(saved) })

	path := filepath.Join(t.TempDir(), "channels.json")
	content := `[{"name":"ok","url":"http://a.com/x.m3u8","input":{"realtime":true,"hlsLiveStartIndex":-5,"hlsSegMaxRetry":3,"hlsMaxReload":100}}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := loadChannelsFromFile(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, ok := GetChannel(0)
	if !ok || ch.Input == nil {
		t.Fatalf("expected channel with input, got %+v", ch)
	}
	if ch.Input.HLSLiveStartIndex == nil || *ch.Input.HLSLiveStartIndex != -5 {
		t.Errorf("HLSLiveStartIndex = %v, want -5", ch.Input.HLSLiveStartIndex)
	}
}

// withConfigFile runs LoadConfig against content written to config.json in a
// temporary working directory, restoring Cfg and the working directory after.
// t.Chdir is unavailable on the module's Go 1.23 baseline.
func withConfigFile(t *testing.T, content string) error {
	t.Helper()
	saved := Cfg
	t.Cleanup(func() { Cfg = saved })

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	if err := os.WriteFile("config.json", []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	Cfg = Config{}
	return LoadConfig()
}

// TestLoadConfigInvalidInput verifies that an invalid config.json input
// option is reported at startup.
func TestLoadConfigInvalidInput(t *testing.T) {
	err := withConfigFile(t, `{"name":"t","device_id":"12345678","input":{"hls_max_reload":-1}}`)
	if err == nil {
		t.Error("expected error for negative hls_max_reload, got nil")
	}
}

// TestLoadConfigValidInput verifies a valid config.json input block loads
// through LoadConfig.
func TestLoadConfigValidInput(t *testing.T) {
	err := withConfigFile(t, `{"name":"t","device_id":"12345678","input":{"realtime":true,"hls_seg_max_retry":3,"hls_max_reload":100}}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if Cfg.Input == nil || Cfg.Input.HLSMaxReload == nil || *Cfg.Input.HLSMaxReload != 100 {
		t.Errorf("Cfg.Input not loaded: %+v", Cfg.Input)
	}
}

func TestInputOptionsHasHLSOptions(t *testing.T) {
	var nilOpts *InputOptions
	if nilOpts.HasHLSOptions() {
		t.Error("nil options should report no HLS options")
	}
	if (&InputOptions{Realtime: boolPtr(true)}).HasHLSOptions() {
		t.Error("realtime alone is not an HLS option")
	}
	if !(&InputOptions{HLSMaxReload: intPtr(100)}).HasHLSOptions() {
		t.Error("HLSMaxReload should count as an HLS option")
	}
}
