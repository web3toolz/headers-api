package main

import (
	cfg "headers-api/config"
	"headers-api/internal/app"
	"log"
)

func main() {
	config, err := cfg.LoadConfigFromYaml("config.yaml")

	if err != nil {
		log.Fatal("failed to load config", config)
	}

	application := app.NewApp(config)

	if err = application.Run(); err != nil {
		log.Fatal("failed to run application", config)
	}
}
