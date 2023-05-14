package main

import (
	"encoding/json"
	"log"
	"os"
	"path"
)

type Config struct {
	Directories []string `json:"directories"`
}

func LoadConfig() *Config {
	content := findConfigFile()
	if content == nil {
		return defaultConfig()
	}

	conf := new(Config)
	err := json.Unmarshal(content, conf)
	if err != nil {
		return defaultConfig()
	}

	return conf
}

func findConfigFile() []byte {
	// searches in $HOME/.config
	configPath := path.Join(mustUserHomeDir(), ".config", "teamus", "teamus.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		content, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatal("Failed to read Config file" + err.Error())
		}
		return content
	}

	// searches in $HOME for config file
	configPath = path.Join(mustUserHomeDir(), ".teamus.json")
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		content, err := os.ReadFile(configPath)
		if err != nil {
			log.Fatal("Failed to read Config file" + err.Error())
		}
		return content
	}

	return nil
}

func defaultConfig() *Config {
	home := mustUserHomeDir()
	dirs := []string{
		path.Join(home, "Music"),
	}
	return &Config{
		Directories: dirs,
	}
}

func mustUserHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Cannot find Home Directory" + err.Error())
	}

	return home
}
