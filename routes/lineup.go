package routes

import (
	"fmt"
	"net/http"

	"plex-dvr-hls/config"
	"github.com/gin-gonic/gin"
)

type ChannelLineup struct {
	GuideNumber string   `json:"GuideNumber"`
	GuideName   string   `json:"GuideName"`
	Tags        []string `json:"Tags"`
	URL         string   `json:"URL"`
}

func Lineup(c *gin.Context) {
	var channelLineups []ChannelLineup

	var host = c.Request.Host

	for index, channel := range config.Channels {
		channelLineups = append(
			channelLineups,
			ChannelLineup{
				GuideNumber: fmt.Sprintf("%d", index+1),
				GuideName:   channel.Name,
				Tags:        make([]string, 0),
				URL:         fmt.Sprintf("http://%s/stream/%d", host, index+1),
			},
		)
	}

	c.JSON(
		http.StatusOK,
		channelLineups,
	)
}

type Status struct {
	ScanInProgress int      `json:"ScanInProgress"`
	ScanPossible   int      `json:"ScanPossible"`
	Source         string   `json:"Source"`
	SourceList     []string `json:"Cable"`
}

func LineupStatus(c *gin.Context) {
	c.JSON(
		http.StatusOK,
		Status{
			ScanInProgress: 0,
			ScanPossible:   1,
			Source:         "Cable",
			SourceList: []string{
				"Cable",
			},
		},
	)

}
