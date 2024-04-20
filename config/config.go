package config

import (
	"encoding/json"
	"log"
	"os"
)

type EncoderProfile string

const (
	EncoderProfileCPU          EncoderProfile = "cpu"
	EncoderProfileVAAPI        EncoderProfile = "vaapi"
	EncoderProfileVideoToolbox EncoderProfile = "video_toolbox"
	EncoderProfileOMX          EncoderProfile = "omx"
)

type Config struct {
	Name           string          `json:"name"`
	EncoderProfile *EncoderProfile `json:"encoder_profile"`
	TunerCount     *int            `json:"tuner_count"`
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
