package config

import (
	"encoding/json"
	"log"
	"os"
)

type ProxyConfig struct {
	Host     string `json:"host"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Channel struct {
	Name        string       `json:"name"`
	URL         string       `json:"url"`
	ProxyConfig *ProxyConfig `json:"proxy"`
}

var (
	Channels []Channel
)

func init() {
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
