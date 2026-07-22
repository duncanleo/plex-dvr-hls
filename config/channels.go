package config

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	m3uparser "github.com/pawanpaudel93/go-m3u-parser/m3uparser"
)

type ProxyConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Channel struct {
	Name                  string       `json:"name"`
	URL                   string       `json:"url"`
	ProxyConfig           *ProxyConfig `json:"proxy"`
	DisableTranscode      bool         `json:"disableTranscode"`
	DisableAudioTranscode bool         `json:"disableAudioTranscode"`
	// Reconnect enables -reconnect_at_eof and -reconnect_streamed for HTTP(S)
	// sources. Off by default: the HLS demuxer manages its own segment cycling
	// and treats a normal playlist-fetch EOF as a reconnect trigger, which
	// causes an infinite loop on providers that 302-redirect each request to a
	// fresh backend. Enable only for direct MPEG-TS or other non-HLS HTTP
	// sources that need low-level reconnect on EOF.
	Reconnect bool `json:"reconnect"`

	// UserAgent is a custom UA string that will be used by FFMPEG to make requests to the stream URL.
	UserAgent *string `json:"userAgent,omitempty"`
	Referer   *string `json:"referer,omitempty"`
	Icon      *string `json:"icon,omitempty"`
}

var (
	channels []Channel
	mu       sync.Mutex
)

// SetChannels replaces the channel list under the mutex.
// Exported so route tests can inject test data without bypassing the lock.
func SetChannels(c []Channel) {
	mu.Lock()
	channels = c
	mu.Unlock()
}

func LoadChannelsFromPl(location string) error {
	var userAgent = os.Getenv("UA")
	if len(userAgent) == 0 {
		userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
	}
	parser := m3uparser.M3uParser{UserAgent: userAgent, Timeout: 60}
	parser.ParseM3u(location, true, true)
	streams := parser.GetStreamsSlice()
	var channels []Channel
	for _, st := range streams {
		title, ok1 := st["title"].(string)
		url, ok2 := st["url"].(string)
		if !ok1 || !ok2 || title == "" || url == "" {
			log.Printf("[PLAYLIST] skipping malformed entry: title=%v url=%v", st["title"], st["url"])
			continue
		}
		channels = append(channels, Channel{Name: title, URL: url, UserAgent: &userAgent})
	}
	if len(channels) == 0 {
		return errors.New("No streams in playlist")
	}
	SetChannels(channels)
	return nil
}

func WatchChannelsFile() {
	backoff := time.Second
	for {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("channels.json watcher setup failed (%v), retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		if err = watcher.Add("channels.json"); err != nil {
			watcher.Close()
			log.Printf("channels.json watcher add failed (%v), retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff = min(backoff*2, 30*time.Second)
			continue
		}

		backoff = time.Second // reset on successful setup
		log.Println("channels.json watcher active")
		runWatchLoop(watcher, LoadChannels)
		watcher.Close()
		log.Println("channels.json watcher restarting...")
	}
}

// runWatchLoop processes fsnotify events until the watcher channel closes.
// reload is injected rather than calling LoadChannels directly so tests can
// assert it was called without touching the filesystem.
func runWatchLoop(watcher *fsnotify.Watcher, reload func() error) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Write covers in-place edits.
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				log.Println("Detected change in channels.json, reloading channels")
				if err := reload(); err != nil {
					log.Printf("Error reloading channels: %s\n", err)
				}
			}
			// Atomic saves (temp-file-then-rename) fire Remove or Rename on the old
			// inode — not Create, which is a directory-level event. Reload first
			// (the new file is already in place), then re-add the watch for the new
			// inode. If re-add fails, return so the outer loop rebuilds a fresh watcher.
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				log.Println("Detected atomic save of channels.json, reloading channels")
				if err := reload(); err != nil {
					log.Printf("Error reloading channels: %s\n", err)
				}
				if err := watcher.Add("channels.json"); err != nil {
					log.Printf("channels.json watcher re-add failed: %v — restarting watcher", err)
					return // outer loop rebuilds a fresh watcher
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %s\n", err)
		}
	}
}

// GetChannel returns the channel at the given zero-based index.
// The index is a volatile slice position, not a stable ID — it changes when
// channels.json is reloaded.
// Safe for concurrent use alongside LoadChannels and LoadChannelsFromPl.
func GetChannel(index int) (Channel, bool) {
	mu.Lock()
	defer mu.Unlock()
	if index < 0 || index >= len(channels) {
		return Channel{}, false
	}
	return channels[index], true
}

// GetChannelCount returns the number of channels without copying the slice.
// Safe for concurrent use alongside LoadChannels and LoadChannelsFromPl.
func GetChannelCount() int {
	mu.Lock()
	defer mu.Unlock()
	return len(channels)
}

// GetChannels returns a snapshot of all channels.
// Safe for concurrent use alongside LoadChannels and LoadChannelsFromPl.
func GetChannels() []Channel {
	mu.Lock()
	defer mu.Unlock()
	result := make([]Channel, len(channels))
	copy(result, channels)
	return result
}

func LoadChannels() error {
	return loadChannelsFromFile("channels.json")
}

func loadChannelsFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var channels []Channel
	if err := json.NewDecoder(file).Decode(&channels); err != nil {
		return err
	}

	if len(channels) == 0 {
		return errors.New("channels.json decoded to empty list — keeping previous channels")
	}

	SetChannels(channels)
	log.Println("Channels reloaded successfully")
	return nil
}

func init() {
	// Skip initialization during tests by checking command line args
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return
		}
	}

	var playlist = os.Getenv("PLAYLIST")
	if len(playlist) > 0 {
		err := LoadChannelsFromPl(playlist)
		if err == nil {
			return
		} else {
			log.Printf("Provided m3u playlist error: %s\n", err)
		}
	}

	err := LoadChannels()
	if err != nil {
		// Fatal here is intentional: with no channels the server cannot serve
		// anything. Contrast with watcher reloads (runWatchLoop) where a failed
		// reload keeps the previous channel list and logs an error — acceptable
		// because the previous list is still valid. At startup there is no
		// previous list to fall back to.
		log.Fatal(err)
	}

	go WatchChannelsFile()
}
