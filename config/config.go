package config

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
)

type EncoderProfile string

const (
	EncoderProfileCPU          EncoderProfile = "cpu"
	EncoderProfileVAAPI        EncoderProfile = "vaapi"
	EncoderProfileVideoToolbox EncoderProfile = "video_toolbox"
	EncoderProfileOMX          EncoderProfile = "omx"
	EncoderProfileNVENC        EncoderProfile = "nvenc"
)

type Config struct {
	Name           string          `json:"name"`
	EncoderProfile *EncoderProfile `json:"encoder_profile"`
	TunerCount     *int            `json:"tuner_count"`
	DeviceID       *string         `json:"device_id"`
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
	case EncoderProfileNVENC:
		return EncoderProfileNVENC
	}

	return EncoderProfileCPU
}

var (
	Cfg Config
)

func LoadConfig() error {
	file, err := os.Open("config.json")
	if err != nil {
		return err
	}
	defer file.Close()

	var decoder = json.NewDecoder(file)
	err = decoder.Decode(&Cfg)
	if err != nil {
		return err
	}

	// Generate a constant device ID if not set in config
	if Cfg.DeviceID == nil {
		deviceID := fmt.Sprintf("%d", rand.Int63n(90000000-10000000)+10000000)
		Cfg.DeviceID = &deviceID
		log.Printf("No device_id in config.json, generated constant ID: %s", deviceID)
	}

	return nil
}

func init() {
	// Skip initialization during tests by checking command line args
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return
		}
	}

	if err := LoadConfig(); err != nil {
		log.Fatal(err)
	}
}
