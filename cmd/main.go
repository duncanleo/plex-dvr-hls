package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"plex-dvr-hls/config"
	"plex-dvr-hls/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	var port = 5004
	var portStr = os.Getenv("PORT")
	var err error

	if len(portStr) > 0 {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			log.Fatal(err)
		}
	}

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.GET("/capability", routes.Capability)
	r.GET("/discover.json", routes.Discover)
	r.GET("/lineup.json", routes.Lineup)
	r.GET("/lineup_status.json", routes.LineupStatus)
	r.GET("/stream/:channelID", routes.Stream)
	r.GET("/xmltv", routes.XMLTV)

	log.Printf("Starting '%s' tuner with encoder profile %s\n", config.Cfg.Name, config.Cfg.GetEncoderProfile())

	r.Run(fmt.Sprintf(":%d", port))
}
