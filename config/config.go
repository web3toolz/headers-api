package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const exampleConfigPath = "config.example.yaml"

type SourceConfig struct {
	Url             string        `json:"url" yaml:"url"`
	Timeout         time.Duration `json:"timeout" yaml:"timeout"`
	PollingInterval time.Duration `json:"pollingInterval" yaml:"pollingInterval"`
}

type Config struct {
	Port    uint
	Sources []*SourceConfig
}

func (c SourceConfig) IsHTTP() bool {
	return strings.HasPrefix(c.Url, "http://") || strings.HasPrefix(c.Url, "https://")
}

func (c SourceConfig) IsWS() bool {
	return strings.HasPrefix(c.Url, "ws://") || strings.HasPrefix(c.Url, "wss://")
}

func openExampleConfig() (*os.File, error) {
	log.Printf("Loading default config (path %s)\n", exampleConfigPath)

	f, err := os.Open(filepath.Clean(exampleConfigPath))

	if err != nil {
		return nil, fmt.Errorf("failed to open example config file: %w", err)
	}

	return f, nil

}

func LoadConfigFromYaml(path string) (*Config, error) {
	config := &Config{}

	f, err := os.Open(filepath.Clean(path))

	if err != nil && os.IsNotExist(err) {
		f, err = openExampleConfig()
	}

	if err != nil {
		return nil, err
	}

	err = yaml.NewDecoder(f).Decode(config)

	if err != nil {
		return nil, err
	}

	return config, nil
}
