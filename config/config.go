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

const deviceIDFile = ".device_id"

func loadDeviceIDFromFile() (string, error) {
	data, err := os.ReadFile(deviceIDFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func saveDeviceIDToFile(deviceID string) error {
	return os.WriteFile(deviceIDFile, []byte(deviceID), 0644)
}

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

	// Device ID priority: config.json > .device_id file > generate new
	if Cfg.DeviceID == nil {
		// Try to load from state file
		if savedID, err := loadDeviceIDFromFile(); err == nil && savedID != "" {
			Cfg.DeviceID = &savedID
			log.Printf("Loaded device_id from %s: %s", deviceIDFile, savedID)
		} else {
			// Generate new ID and save it
			deviceID := fmt.Sprintf("%d", rand.Int63n(90000000-10000000)+10000000)
			Cfg.DeviceID = &deviceID

			// Try to save to file, but don't fail if we can't (e.g., read-only filesystem)
			if err := saveDeviceIDToFile(deviceID); err != nil {
				log.Printf("Warning: Generated device_id %s but could not save to %s: %v", deviceID, deviceIDFile, err)
				log.Printf("Device ID will change on restart. Set device_id in config.json to make it permanent.")
			} else {
				log.Printf("Generated and saved device_id to %s: %s", deviceIDFile, deviceID)
			}
		}
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
