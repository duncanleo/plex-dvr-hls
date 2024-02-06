package config

import (
	"encoding/json"
	"log"
	"os"
	"errors"
	m3uparser "github.com/pawanpaudel93/go-m3u-parser/m3uparser"
)

type ProxyConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Channel struct {
	Name             string       `json:"name"`
	URL              string       `json:"url"`
	ProxyConfig      *ProxyConfig `json:"proxy"`
	DisableTranscode bool         `json:"disableTranscode"`

	// UserAgent is a custom UA string that will be used by FFMPEG to make requests to the stream URL.
	UserAgent *string `json:"userAgent,omitempty"`
}

var (
	Channels []Channel
)

func LoadChannelsFromPl(location string) error {
    var userAgent = os.Getenv("UA")
    if len(userAgent) == 0 {
      userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36"
	}
	parser := m3uparser.M3uParser{UserAgent: userAgent, Timeout: 60}
	parser.ParseM3u(location,true,true)
	streams := parser.GetStreamsSlice()
	if len(streams) > 0 {
	    for _, st := range streams {
			ch := Channel{Name: st["title"].(string), URL: st["url"].(string), UserAgent: &userAgent}
			Channels = append(Channels,ch)
		} 
		if len(Channels) > 0 {
			return nil
		}
	}
	return errors.New("No streams in playlist")
}

func init() {
    var playlist = os.Getenv("PLAYLIST")
	if len(playlist) > 0 {
	    err := LoadChannelsFromPl(playlist)	
	    if err == nil {
			return
		} else {
		    log.Printf("Provided m3u playlist error: %s\n",err)
		}
	}

	file, err := os.Open("channels.json")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	var decoder = json.NewDecoder(file)
	err = decoder.Decode(&Channels)

	if err != nil {
		log.Fatal(err)
	}
}
