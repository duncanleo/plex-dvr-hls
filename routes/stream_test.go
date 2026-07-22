package routes

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/duncanleo/plex-dvr-hls/config"
	"github.com/gin-gonic/gin"
)

// fakeExecCommand returns a no-op OS command that exits immediately with zero
// stdout bytes, simulating an ffmpeg process that fails to produce any output.
func fakeExecCommand(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", "exit")
	}
	return exec.CommandContext(ctx, "sh", "-c", "exit")
}

// TestStreamPipeEOFReturnsNilError documents that io.Copy on a closed pipe
// returns nil — not io.EOF. This is why the stream retry loop treats n==0 as
// a failure signal: a clean ffmpeg exit (no output written) is indistinguishable
// from a successful-but-empty read via the error value alone.
func TestStreamPipeEOFReturnsNilError(t *testing.T) {
	pr, pw := io.Pipe()
	pw.Close()

	var buf bytes.Buffer
	_, err := io.Copy(&buf, pr)
	if err != nil {
		t.Errorf("io.Copy on closed pipe: expected nil, got %v", err)
	}
}

// TestStreamHandlerBadRequest verifies Stream returns 400 for a non-numeric channel ID.
func TestStreamHandlerBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/abc", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric channel ID, got %d", w.Code)
	}
}

// TestStreamHandlerNotFound verifies Stream returns 404 when the channel index
// is out of range.
func TestStreamHandlerNotFound(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{})

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/99", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown channel, got %d", w.Code)
	}
}

// TestStreamRetryLoop502 verifies that when ffmpeg exits immediately without
// writing any bytes, the handler retries streamMaxFailures times then returns 502.
func TestStreamRetryLoop502(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	origCmd := execCommandContext
	execCommandContext = fakeExecCommand
	t.Cleanup(func() { execCommandContext = origCmd })

	origDelay := streamRetryDelay
	streamRetryDelay = 5 * time.Millisecond
	t.Cleanup(func() { streamRetryDelay = origDelay })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 after %d zero-byte exits, got %d", streamMaxFailures, w.Code)
	}
}

// fakeExecCommandSlow returns a command that runs for longer than the default
// minHealthyDuration (10s), used to verify that a healthy run resets quickRestarts.
// Uses sleep directly (no shell wrapper) so exec.CommandContext can kill it cleanly.
func fakeExecCommandSlow(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Start-Sleep -Milliseconds 200")
	}
	return exec.CommandContext(ctx, "sleep", "0.2")
}

// TestStreamRetryLoopHealthyReset verifies that a run lasting longer than
// minHealthyDuration resets quickRestarts to zero. Without the reset, a sign
// flip on the < vs >= comparison would be undetectable by the other tests.
func TestStreamRetryLoopHealthyReset(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	// Call sequence: (streamMaxFailures-1) quick → 1 healthy (reset) → streamMaxFailures quick → 502
	// Total calls: 2 * streamMaxFailures
	var (
		callMu    sync.Mutex
		callCount int
	)
	origCmd := execCommandContext
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		callMu.Lock()
		n := callCount
		callCount++
		callMu.Unlock()
		if n == streamMaxFailures-1 {
			return fakeExecCommandSlow(ctx, name, arg...)
		}
		return fakeExecCommand(ctx, name, arg...)
	}
	t.Cleanup(func() { execCommandContext = origCmd })

	origDelay := streamRetryDelay
	streamRetryDelay = 5 * time.Millisecond
	t.Cleanup(func() { streamRetryDelay = origDelay })

	origMin := minHealthyDuration
	minHealthyDuration = 100 * time.Millisecond
	t.Cleanup(func() { minHealthyDuration = origMin })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502 after reset+cap cycle, got %d", w.Code)
	}
	callMu.Lock()
	total := callCount
	callMu.Unlock()
	expected := 2 * streamMaxFailures
	if total != expected {
		t.Errorf("expected %d total calls (%d quick + 1 healthy reset + %d quick), got %d — healthy-run reset may not be working",
			expected, streamMaxFailures-1, streamMaxFailures, total)
	}
}

// TestStreamStartFailureReturns500 verifies that when ffmpeg fails to start
// (e.g. binary not found), the handler returns 500 and does not retry.
func TestStreamStartFailureReturns500(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	origCmd := execCommandContext
	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "nonexistent-binary-that-does-not-exist-xyz")
	}
	t.Cleanup(func() { execCommandContext = origCmd })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 on ffmpeg start failure, got %d", w.Code)
	}
}

// TestBuildFFmpegArgsEncoderProfiles verifies each hwaccel profile produces
// the correct pre-input and codec args.
func TestBuildFFmpegArgsEncoderProfiles(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "http://example.com"}

	tests := []struct {
		profile      config.EncoderProfile
		wantPreInput []string
		wantCodec    string
	}{
		{
			profile:      config.EncoderProfileCPU,
			wantPreInput: nil,
			wantCodec:    "libx264",
		},
		{
			profile:      config.EncoderProfileVAAPI,
			wantPreInput: []string{"-vaapi_device", "-hwaccel", "vaapi", "-hwaccel_output_format"},
			wantCodec:    "h264_vaapi",
		},
		{
			profile:      config.EncoderProfileVideoToolbox,
			wantPreInput: []string{"-hwaccel", "videotoolbox"},
			wantCodec:    "h264_videotoolbox",
		},
		{
			profile:      config.EncoderProfileNVENC,
			wantPreInput: []string{"-hwaccel", "cuda", "-hwaccel_output_format"},
			wantCodec:    "h264_nvenc",
		},
		{
			profile:      config.EncoderProfileOMX,
			wantPreInput: nil,
			wantCodec:    "h264_omx",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			config.Cfg.EncoderProfile = &tt.profile
			defer func() { config.Cfg.EncoderProfile = nil }()

			args := buildFFmpegArgs(channel, "")
			iIdx := findInputIdx(args)
			if iIdx == -1 {
				t.Fatalf("profile %s: -i flag not found in args", tt.profile)
			}

			for _, flag := range tt.wantPreInput {
				if !contains(args[:iIdx], flag) {
					t.Errorf("profile %s: expected arg %q before -i", tt.profile, flag)
				}
			}
			if !contains(args[iIdx:], tt.wantCodec) {
				t.Errorf("profile %s: expected codec %q after -i", tt.profile, tt.wantCodec)
			}
		})
	}
}

// TestBuildFFmpegArgsDisableTranscode verifies that disableTranscode uses
// -c:v copy instead of a software/hardware encoder.
func TestBuildFFmpegArgsDisableTranscode(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "http://example.com", DisableTranscode: true}
	args := buildFFmpegArgs(channel, "")
	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	postInput := args[iIdx:]

	if !contains(postInput, "copy") {
		t.Error("disableTranscode: expected -c:v copy after -i")
	}
	for _, enc := range []string{"libx264", "h264_vaapi", "h264_nvenc", "h264_videotoolbox", "h264_omx"} {
		if contains(args, enc) {
			t.Errorf("disableTranscode: unexpected encoder %q", enc)
		}
	}
}

// TestBuildFFmpegArgsHeaders verifies UserAgent and Referer are combined into
// a single -headers value so neither overwrites the other in ffmpeg's header dict.
func TestBuildFFmpegArgsHeaders(t *testing.T) {
	ua := "TestAgent/1.0"
	ref := "https://example.com"
	channel := config.Channel{
		Name:      "test",
		URL:       "http://example.com",
		UserAgent: &ua,
		Referer:   &ref,
	}
	args := buildFFmpegArgs(channel, "")
	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	preInput := args[:iIdx]

	// Count -headers occurrences before -i — must be exactly one combined value.
	headerCount := 0
	combinedValue := ""
	for i, a := range preInput {
		if a == "-headers" && i+1 < len(preInput) {
			headerCount++
			combinedValue = preInput[i+1]
		}
	}
	if headerCount != 1 {
		t.Errorf("expected exactly 1 -headers flag before -i, got %d (second would silently overwrite the first)", headerCount)
	}
	if !strings.Contains(combinedValue, "User-Agent: "+ua) {
		t.Errorf("UserAgent missing from combined -headers value: %q", combinedValue)
	}
	if !strings.Contains(combinedValue, "Referer: "+ref) {
		t.Errorf("Referer missing from combined -headers value: %q", combinedValue)
	}
}

// TestBuildFFmpegArgsHeadersPartial verifies that setting only one of UserAgent
// or Referer still produces exactly one -headers flag, and that setting neither
// produces no -headers flag at all.
func TestBuildFFmpegArgsHeadersPartial(t *testing.T) {
	ua := "TestAgent/1.0"
	ref := "https://example.com"

	countHeadersBeforeI := func(args []string) int {
		iIdx := findInputIdx(args)
		if iIdx == -1 {
			iIdx = len(args)
		}
		n := 0
		for _, a := range args[:iIdx] {
			if a == "-headers" {
				n++
			}
		}
		return n
	}

	// UA only
	args := buildFFmpegArgs(config.Channel{Name: "t", URL: "http://x.com", UserAgent: &ua}, "")
	if countHeadersBeforeI(args) != 1 {
		t.Errorf("UA-only: expected 1 -headers flag before -i, got %d", countHeadersBeforeI(args))
	}
	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("UA-only: -i flag not found")
	}
	if !contains(args[:iIdx], "User-Agent: "+ua+"\r\n") {
		t.Errorf("UA-only: UA missing before -i")
	}

	// Referer only
	args = buildFFmpegArgs(config.Channel{Name: "t", URL: "http://x.com", Referer: &ref}, "")
	if countHeadersBeforeI(args) != 1 {
		t.Errorf("Referer-only: expected 1 -headers flag before -i, got %d", countHeadersBeforeI(args))
	}
	iIdx = findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("Referer-only: -i flag not found")
	}
	if !contains(args[:iIdx], "Referer: "+ref+"\r\n") {
		t.Errorf("Referer-only: Referer missing before -i")
	}

	// Neither
	args = buildFFmpegArgs(config.Channel{Name: "t", URL: "http://x.com"}, "")
	if countHeadersBeforeI(args) != 0 {
		t.Errorf("no headers: expected 0 -headers flags before -i, got %d", countHeadersBeforeI(args))
	}
}

// TestBuildFFmpegArgsProxy verifies proxy config is passed as -http_proxy.
func TestBuildFFmpegArgsProxy(t *testing.T) {
	channel := config.Channel{
		Name: "test",
		URL:  "http://example.com",
		ProxyConfig: &config.ProxyConfig{
			Host:     "proxy.example.com:3128",
			Username: "user",
			Password: "pass",
		},
	}
	args := buildFFmpegArgs(channel, "")
	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	preInput := args[:iIdx]

	if !contains(preInput, "-http_proxy") {
		t.Error("proxy: expected -http_proxy flag before -i")
	}
	if !contains(preInput, "http://user:pass@proxy.example.com:3128") {
		t.Error("proxy: expected formatted proxy URL before -i")
	}
}

// fakeExecCommandWriteAndExit returns a command that writes a few bytes to
// stdout then exits — simulating an ffmpeg process that starts successfully but
// dies almost immediately (the bytes-then-die flapping pattern).
func fakeExecCommandWriteAndExit(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/c", "echo", "x")
	}
	return exec.CommandContext(ctx, "sh", "-c", "printf x")
}

// fakeExecCommandBlock returns a command that blocks until its context is
// cancelled, simulating an ffmpeg process that is running a live stream.
// On Windows, powershell Start-Sleep is used because Git Bash shadows the
// built-in timeout.exe with its own version that rejects Windows-style flags.
func fakeExecCommandBlock(ctx context.Context, name string, arg ...string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", "Start-Sleep 30")
	}
	// Direct exec (no sh wrapper) so exec.CommandContext's SIGKILL reaches the
	// sleep process itself — a sh grandchild would survive the kill and hold the
	// pipe open, preventing io.Copy from returning.
	return exec.CommandContext(ctx, "sleep", "30")
}

// TestStreamRetryLoopBytesAndDie verifies that an ffmpeg process that writes
// bytes but exits quickly (short-lived) still accumulates toward the restart
// cap. Previously consecutiveFailures only incremented on zero-byte exits, so
// the 1-byte / 0-byte alternating pattern could bypass the cap entirely.
func TestStreamRetryLoopBytesAndDie(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	origCmd := execCommandContext
	execCommandContext = fakeExecCommandWriteAndExit
	t.Cleanup(func() { execCommandContext = origCmd })

	origDelay := streamRetryDelay
	streamRetryDelay = 5 * time.Millisecond
	t.Cleanup(func() { streamRetryDelay = origDelay })
	// minHealthyDuration left at default (10s): any sub-second process is "quick".

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/stream/1", nil)

	// Once bytes are written the response is already 200, so we can't assert 502.
	// What matters is that the handler returns (cap fired) rather than looping forever.
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Errorf("handler did not return after %d short-lived exits — bytes-then-die pattern bypasses cap", streamMaxFailures)
	}
	if w.Body.Len() == 0 {
		t.Error("expected bytes in response body before handler gave up")
	}
}

// TestStreamContextCancellationKillsFFmpeg verifies that cancelling the
// request context (client disconnect or server shutdown) causes the handler
// to return promptly even while ffmpeg is blocking on output.
func TestStreamContextCancellationKillsFFmpeg(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	origCmd := execCommandContext
	execCommandContext = fakeExecCommandBlock
	t.Cleanup(func() { execCommandContext = origCmd })

	origDelay := streamRetryDelay
	streamRetryDelay = 5 * time.Millisecond
	t.Cleanup(func() { streamRetryDelay = origDelay })

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(context.Background()))

	w := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/stream/1", nil)

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("handler did not exit within 2s after context cancellation")
		<-done // wait for goroutine to finish before t.Cleanup mutates execCommandContext
	}
}

// TestStreamServerCtxCancellationKillsFFmpeg verifies that cancelling the
// serverCtx (SIGTERM path in main.go) kills in-flight streams. Unlike the
// client-disconnect test above, this uses a non-cancelled request context so
// only the server-side path is exercised.
func TestStreamServerCtxCancellationKillsFFmpeg(t *testing.T) {
	t.Cleanup(func() { config.SetChannels(nil) })
	config.SetChannels([]config.Channel{{Name: "test", URL: "http://example.com"}})

	origCmd := execCommandContext
	execCommandContext = fakeExecCommandBlock
	t.Cleanup(func() { execCommandContext = origCmd })

	origDelay := streamRetryDelay
	streamRetryDelay = 5 * time.Millisecond
	t.Cleanup(func() { streamRetryDelay = origDelay })

	gin.SetMode(gin.TestMode)
	serverCtx, serverCancel := context.WithCancel(context.Background())
	router := gin.New()
	router.GET("/stream/:channelID", StreamWithContext(serverCtx))

	w := httptest.NewRecorder()
	// Request context is never cancelled — only serverCtx will be.
	req, _ := http.NewRequest(http.MethodGet, "/stream/1", nil)

	done := make(chan struct{})
	go func() {
		router.ServeHTTP(w, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	serverCancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("handler did not exit within 2s after server context cancellation")
		<-done
	}
}

func contains(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}

// findInputIdx returns the index of "-i" in args, or -1 if not found.
// Used to assert that flags appear in the correct position relative to the
// input source: pre-input opts (proxy, headers, hwaccel) must precede -i;
// codec/filter opts must follow it.
func findInputIdx(args []string) int {
	for i, a := range args {
		if a == "-i" {
			return i
		}
	}
	return -1
}

// TestBuildFFmpegArgsReconnect verifies that -reconnect is present before -i
// for HTTP(S) sources so ffmpeg recovers from true connection drops.
func TestBuildFFmpegArgsReconnect(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "http://example.com/stream.m3u8"}
	args := buildFFmpegArgs(channel, "")

	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	preInput := args[:iIdx]

	if !contains(preInput, "-reconnect") {
		t.Error("-reconnect missing before -i — stream will not recover from upstream drops")
	}
}

// TestBuildFFmpegArgsReconnectAtEofNotDefault confirms that -reconnect_at_eof
// and -reconnect_streamed are absent by default (see Channel.Reconnect doc).
func TestBuildFFmpegArgsReconnectAtEofNotDefault(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "https://example.com/live.m3u8"}
	args := buildFFmpegArgs(channel, "")
	for _, flag := range []string{"-reconnect_at_eof", "-reconnect_streamed"} {
		if contains(args, flag) {
			t.Errorf("flag %q must not be present by default — breaks HLS streams with rotating redirect targets", flag)
		}
	}
}

// TestBuildFFmpegArgsReconnectOptIn verifies that a channel with Reconnect:true
// gets the full set of reconnect flags including -reconnect_at_eof and
// -reconnect_streamed, for sources (e.g. direct MPEG-TS over HTTP) that need them.
func TestBuildFFmpegArgsReconnectOptIn(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "https://example.com/stream.ts", Reconnect: true}
	args := buildFFmpegArgs(channel, "")
	iIdx := findInputIdx(args)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	preInput := args[:iIdx]
	for _, flag := range []string{"-reconnect", "-reconnect_at_eof", "-reconnect_streamed", "-reconnect_delay_max"} {
		if !contains(preInput, flag) {
			t.Errorf("flag %q missing before -i for channel with Reconnect:true", flag)
		}
	}
}

// TestBuildFFmpegArgsReconnectRTSP verifies that -reconnect* flags are NOT
// added for non-HTTP URLs (e.g. RTSP). These flags are HTTP-only; ffmpeg
// errors on them for other protocols, which would register as a quick-exit and
// give up with 502 after streamMaxFailures retries.
func TestBuildFFmpegArgsReconnectRTSP(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "rtsp://camera.local/stream"}
	args := buildFFmpegArgs(channel, "")
	for _, flag := range []string{"-reconnect", "-reconnect_at_eof", "-reconnect_streamed", "-reconnect_delay_max"} {
		if contains(args, flag) {
			t.Errorf("RTSP URL: reconnect flag %q must not be present (HTTP-only option)", flag)
		}
	}
}

// TestBuildFFmpegArgsReconnectOptInRTSP verifies that Reconnect:true on an RTSP
// channel still produces no reconnect flags — the HTTP(S) URL guard is the
// sole gate, not the Reconnect field alone.
func TestBuildFFmpegArgsReconnectOptInRTSP(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "rtsp://camera.local/stream", Reconnect: true}
	args := buildFFmpegArgs(channel, "")
	for _, flag := range []string{"-reconnect", "-reconnect_at_eof", "-reconnect_streamed", "-reconnect_delay_max"} {
		if contains(args, flag) {
			t.Errorf("RTSP URL with Reconnect:true: flag %q must not be present — HTTP(S) guard must be the sole gate", flag)
		}
	}
}

// TestBuildFFmpegArgsAudioTranscode verifies disableAudioTranscode switches
// from re-encoding audio at 256k to passing the source audio through unchanged.
func TestBuildFFmpegArgsAudioTranscode(t *testing.T) {
	reencoded := buildFFmpegArgs(config.Channel{Name: "test", URL: "http://example.com"}, "")
	iIdx := findInputIdx(reencoded)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	postInput := reencoded[iIdx:]
	if !contains(postInput, "-b:a") {
		t.Error("default: expected -b:a after -i for audio re-encoding")
	}
	if contains(postInput, "-c:a") {
		t.Error("default: unexpected -c:a copy after -i")
	}

	copied := buildFFmpegArgs(config.Channel{Name: "test", URL: "http://example.com", DisableAudioTranscode: true}, "")
	iIdx = findInputIdx(copied)
	if iIdx == -1 {
		t.Fatal("-i flag not found in args")
	}
	postInput = copied[iIdx:]
	if contains(postInput, "-b:a") {
		t.Error("disableAudioTranscode: unexpected -b:a after -i, audio should be copied")
	}
	if !contains(postInput, "-c:a") {
		t.Error("disableAudioTranscode: expected -c:a copy after -i")
	}
}

// TestBuildFFmpegArgsMobile verifies that both "mobile" and "internet720"
// transcode modes apply resolution scaling. Previously "mobile" matched an
// empty switch case and produced no scaling args while "internet720" did.
func TestBuildFFmpegArgsMobile(t *testing.T) {
	channel := config.Channel{Name: "test", URL: "http://example.com"}

	hasScaling := func(args []string) bool {
		for i, a := range args {
			if a == "-s" && i+1 < len(args) && args[i+1] == "1280x720" {
				return true
			}
		}
		return false
	}

	if !hasScaling(buildFFmpegArgs(channel, "mobile")) {
		t.Error("mobile transcode missing -s 1280x720")
	}
	if !hasScaling(buildFFmpegArgs(channel, "internet720")) {
		t.Error("internet720 transcode missing -s 1280x720")
	}
	if hasScaling(buildFFmpegArgs(channel, "")) {
		t.Error("default transcode should not include -s 1280x720")
	}
}
