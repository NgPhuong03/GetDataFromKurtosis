package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Mongo MongoConfig
	Dora DoraConfig
	TimeLoop int `yaml:"time_loop"`
}

type MongoConfig struct {
	MongoURI string `yaml:"mongoURI"`
	DBName   string `yaml:"dbName"`
}

type DoraConfig struct {
	URL string `yaml:"url"`
}

func LoadConfig() *Config {
	yamlFile, err := os.ReadFile("config/config.yaml")
	if err != nil {
		log.Fatalf("failed to read config file: %v", err)
	}
	var config Config
	yaml.Unmarshal(yamlFile, &config)
	return &config
}