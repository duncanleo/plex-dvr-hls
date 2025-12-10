package routes

import (
	"fmt"
	"net/http"

	"github.com/duncanleo/plex-dvr-hls/config"
	"github.com/gin-gonic/gin"
)

type DVR struct {
	FriendlyName    string `json:"FriendlyName"`
	ModelNumber     string `json:"ModelNumber"`
	FirmwareName    string `json:"FirmwareName"`
	TunerCount      int    `json:"TunerCount"`
	FirmwareVersion string `json:"FirmwareVersion"`
	DeviceID        string `json:"DeviceID"`
	DeviceAuth      string `json:"DeviceAuth"`
	BaseURL         string `json:"BaseURL"`
	LineupURL       string `json:"LineupURL"`
	Manufacturer    string `json:"Manufacturer"`
}

func Discover(c *gin.Context) {
	var host = c.Request.Host

	// Defaults to channel count * 3
	var tunerCount = len(config.Channels) * 3
	if config.Cfg.TunerCount != nil {
		tunerCount = *config.Cfg.TunerCount
	}

	c.JSON(
		http.StatusOK,
		DVR{
			FriendlyName:    config.Cfg.Name,
			ModelNumber:     "HDTC-2US",
			FirmwareName:    "hdhomeruntc_atsc",
			TunerCount:      tunerCount,
			FirmwareVersion: "20150826",
			DeviceID:        *config.Cfg.DeviceID,
			DeviceAuth:      "test1234",
			BaseURL:         fmt.Sprintf("http://%s", host),
			LineupURL:       fmt.Sprintf("http://%s/lineup.json", host),
			Manufacturer:    "Silicondust",
		},
	)
}
