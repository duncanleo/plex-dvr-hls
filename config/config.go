package config

import (
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Name string `json:"name"`
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
