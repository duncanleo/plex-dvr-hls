package config

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
)

type EncoderProfile string

const (
	EncoderProfileCPU          EncoderProfile = "cpu"
	EncoderProfileVAAPI        EncoderProfile = "vaapi"
	EncoderProfileVideoToolbox EncoderProfile = "video_toolbox"
	EncoderProfileOMX          EncoderProfile = "omx"
)

type PlexServer struct {
    Endpoint string `json:"endpoint"`
    Token    string `json:"token"`
}

type Config struct {
	Name           string          		`json:"name"`
	EncoderProfile *EncoderProfile 		`json:"encoder_profile"`
	TunerCount     *int              	`json:"tuner_count"`
	PlexServers    []PlexServer 	   	`json:"plex_servers"`
}

func (c Config) GetEncoderProfile() EncoderProfile {
	if c.EncoderProfile == nil {
		return EncoderProfileCPU
	}

	switch *c.EncoderProfile {
	case EncoderProfileVAAPI:
		return EncoderProfileVAAPI
	case EncoderProfileOMX:
		return EncoderProfileOMX
	case EncoderProfileVideoToolbox:
		return EncoderProfileVideoToolbox
	}

	return EncoderProfileCPU
}

var (
	Cfg Config
)

func RefreshPlexLiveTVGuide() error {
    for _, server := range Cfg.PlexServers {
        url := server.Endpoint + "/livetv/dvrs/1/guide?X-Plex-Token=" + server.Token

        req, err := http.NewRequest("PUT", url, nil)
        if err != nil {
            return err
        }

        client := &http.Client{}
        resp, err := client.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()

        if resp.StatusCode != http.StatusOK {
            return fmt.Errorf("failed to refresh Plex LiveTV Guide for server %s, status code: %d", server.Endpoint, resp.StatusCode)
        }

        log.Printf("Plex LiveTV Guide refreshed successfully for server %s", server.Endpoint)
    }
    return nil
}

func init() {
	file, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	var decoder = json.NewDecoder(file)
	err = decoder.Decode(&Cfg)

	if err != nil {
		log.Fatal(err)
	}
}
