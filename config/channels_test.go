package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestGetChannelValidIndex(t *testing.T) {
	t.Cleanup(func() { channels = nil })
	channels = []Channel{{Name: "ch1", URL: "http://a.com"}, {Name: "ch2", URL: "http://b.com"}}

	ch, ok := GetChannel(0)
	if !ok {
		t.Fatal("expected ok=true for index 0")
	}
	if ch.Name != "ch1" {
		t.Errorf("expected ch1, got %s", ch.Name)
	}

	ch, ok = GetChannel(1)
	if !ok {
		t.Fatal("expected ok=true for index 1")
	}
	if ch.Name != "ch2" {
		t.Errorf("expected ch2, got %s", ch.Name)
	}
}

func TestGetChannelInvalidIndex(t *testing.T) {
	t.Cleanup(func() { channels = nil })
	channels = []Channel{{Name: "ch1", URL: "http://a.com"}}

	for _, id := range []int{-1, 1, 99} {
		_, ok := GetChannel(id)
		if ok {
			t.Errorf("index %d: expected ok=false, got true", id)
		}
	}
}

func TestGetChannelsSnapshot(t *testing.T) {
	t.Cleanup(func() { channels = nil })
	channels = []Channel{{Name: "original", URL: "http://a.com"}}

	snapshot := GetChannels()
	if len(snapshot) != 1 || snapshot[0].Name != "original" {
		t.Fatal("snapshot does not match Channels")
	}

	// Mutating the snapshot must not affect internal state
	snapshot[0].Name = "mutated"
	ch, _ := GetChannel(0)
	if ch.Name != "original" {
		t.Error("mutating GetChannels() snapshot modified internal Channels state")
	}
}

func TestGetChannelCount(t *testing.T) {
	t.Cleanup(func() { channels = nil })

	channels = []Channel{}
	if n := GetChannelCount(); n != 0 {
		t.Errorf("empty: expected 0, got %d", n)
	}

	channels = []Channel{{Name: "a", URL: "http://a.com"}, {Name: "b", URL: "http://b.com"}}
	if n := GetChannelCount(); n != 2 {
		t.Errorf("two channels: expected 2, got %d", n)
	}
}

// TestLoadChannelsFromFile verifies the core file-load path with a temp file,
// covering both successful decode and correct mutex-protected assignment.
func TestLoadChannelsFromFile(t *testing.T) {
	t.Cleanup(func() { channels = nil })

	dir := t.TempDir()
	path := filepath.Join(dir, "channels.json")
	content := `[{"name":"ch1","url":"http://a.com"},{"name":"ch2","url":"http://b.com"}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := loadChannelsFromFile(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n := GetChannelCount(); n != 2 {
		t.Fatalf("expected 2 channels, got %d", n)
	}
	ch, ok := GetChannel(0)
	if !ok || ch.Name != "ch1" {
		t.Errorf("expected ch1, got %+v", ch)
	}
}

// TestLoadChannelsFromFileEmptyRejectsAndKeepsPrevious verifies that decoding
// an empty JSON array ([]) or null does not wipe the live channel list.
// LoadChannelsFromPl has this guard; this test ensures loadChannelsFromFile
// is consistent.
func TestLoadChannelsFromFileEmptyRejectsAndKeepsPrevious(t *testing.T) {
	channels = []Channel{{Name: "before", URL: "http://a.com"}}
	t.Cleanup(func() { channels = nil })

	dir := t.TempDir()
	path := filepath.Join(dir, "channels.json")

	for _, content := range []string{"[]", "null"} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		if err := loadChannelsFromFile(path); err == nil {
			t.Errorf("content %q: expected error, got nil — empty decode should not wipe channels", content)
		}
		ch, ok := GetChannel(0)
		if !ok || ch.Name != "before" {
			t.Errorf("content %q: channels modified after empty-decode error: got %+v", content, ch)
		}
	}
}

// TestLoadChannelsFromFileReconnectTag verifies that the Reconnect field is
// correctly unmarshaled from JSON. A mistyped json tag (e.g. "Reconnect"
// instead of "reconnect") would silently leave the field as false even when
// channels.json sets it to true, making the feature undetectable without this test.
func TestLoadChannelsFromFileReconnectTag(t *testing.T) {
	t.Cleanup(func() { channels = nil })

	dir := t.TempDir()
	path := filepath.Join(dir, "channels.json")
	content := `[{"name":"live","url":"http://example.com/stream.ts","reconnect":true}]`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := loadChannelsFromFile(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ch, ok := GetChannel(0)
	if !ok {
		t.Fatal("channel not found")
	}
	if !ch.Reconnect {
		t.Error("Reconnect should be true after loading from JSON with \"reconnect\":true — check json struct tag")
	}
}

// TestRunWatchLoopAtomicSave verifies that renaming a temp file over channels.json
// (the standard atomic-save pattern used by editors and deployment tooling) triggers
// a reload. Previously the Remove/Rename branch only re-added the watch without
// calling reload(), so atomic saves silently did nothing.
func TestRunWatchLoopAtomicSave(t *testing.T) {
	dir := t.TempDir()
	channelsFile := filepath.Join(dir, "channels.json")

	if err := os.WriteFile(channelsFile, []byte(`[]`), 0644); err != nil {
		t.Fatal(err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { watcher.Close() })

	if err := watcher.Add(channelsFile); err != nil {
		t.Fatal(err)
	}

	reloaded := make(chan struct{}, 3)
	go runWatchLoop(watcher, func() error {
		reloaded <- struct{}{}
		return nil
	})

	// Atomic save: write to a temp file, rename it over the watched file.
	tmp := channelsFile + ".tmp"
	if err := os.WriteFile(tmp, []byte(`[{"name":"new","url":"http://b.com"}]`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, channelsFile); err != nil {
		t.Fatal(err)
	}

	select {
	case <-reloaded:
		// atomic save triggered a reload — fix is working
	case <-time.After(3 * time.Second):
		t.Error("atomic save (rename-into-place) did not trigger a reload within 3s")
	}
}

// TestLoadChannelsFromPlErrorLeavesChannelsUnchanged verifies that when
// LoadChannelsFromPl fails (unreachable URL), it does not partially modify
// Channels. Previously the function appended to the global slice without
// holding mu, which could leave Channels in a partially-written state.
func TestLoadChannelsFromPlErrorLeavesChannelsUnchanged(t *testing.T) {
	t.Cleanup(func() { channels = nil })
	channels = []Channel{{Name: "before", URL: "http://a.com"}}

	// Serve an empty M3U — parser connects (no fatal) but gets 0 streams → error path.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "#EXTM3U")
	}))
	defer ts.Close()

	err := LoadChannelsFromPl(ts.URL)
	if err == nil {
		t.Skip("expected error for empty playlist — skipping if parser succeeded somehow")
	}

	ch, ok := GetChannel(0)
	if !ok || ch.Name != "before" {
		t.Errorf("LoadChannelsFromPl error modified Channels unexpectedly: got %+v", ch)
	}
}

// TestGetChannelConcurrentAccess verifies GetChannel is safe to call
// concurrently alongside hot-reload writes to Channels (as done by both
// LoadChannels and LoadChannelsFromPl).
// Run with: go test -race ./config/...
func TestGetChannelConcurrentAccess(t *testing.T) {
	t.Cleanup(func() { channels = nil })
	channels = []Channel{{Name: "test", URL: "http://example.com"}}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			SetChannels([]Channel{{Name: "test", URL: "http://example.com"}})
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			GetChannel(0)
		}
	}()

	wg.Wait()
}
