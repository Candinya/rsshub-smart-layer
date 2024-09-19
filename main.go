package main

import (
	"flag"
	"log"
	"os"

	"github.com/candinya/rsshub-smart-layer/app"
	"github.com/candinya/rsshub-smart-layer/types"
	"gopkg.in/yaml.v3"
)

var configPath string

func init() {
	flag.StringVar(&configPath, "config", "config.yml", "path to config file")
}

func main() {
	flag.Parse()

	// Read config
	configFileBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}

	// Parse config
	var cfg types.Config
	err = yaml.Unmarshal(configFileBytes, &cfg)
	if err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}

	// Start application
	err = app.Start(&cfg)
	if err != nil {
		log.Printf("app stop: %v", err)
	}
}
