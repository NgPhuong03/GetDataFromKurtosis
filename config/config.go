package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Mongo         MongoConfig `yaml:"mongo"`
	Dora          DoraConfig  `yaml:"dora"`
	Model         string      `yaml:"model"`
	SlotsPerEpoch int         `yaml:"slots_per_epoch"`
	TargetEpochs  int         `yaml:"target_epochs"`
	OutputDir     string      `yaml:"output_dir"`
	TimeLoop      int         `yaml:"time_loop"`
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
	var cfg Config
	if err := yaml.Unmarshal(yamlFile, &cfg); err != nil {
		log.Fatalf("failed to parse config file: %v", err)
	}
	applyEnv(&cfg)
	if err := cfg.normalize(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	return &cfg
}

func applyEnv(cfg *Config) {
	if v := strings.TrimSpace(os.Getenv("MODEL")); v != "" {
		cfg.Model = v
	}
	if v := strings.TrimSpace(os.Getenv("DORA_URL")); v != "" {
		cfg.Dora.URL = v
	}
	if v := strings.TrimSpace(os.Getenv("OUTPUT_DIR")); v != "" {
		cfg.OutputDir = v
	}
	if v := strings.TrimSpace(os.Getenv("TARGET_EPOCHS")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid TARGET_EPOCHS: %v", err)
		}
		cfg.TargetEpochs = n
	}
	if v := strings.TrimSpace(os.Getenv("SLOTS_PER_EPOCH")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			log.Fatalf("invalid SLOTS_PER_EPOCH: %v", err)
		}
		cfg.SlotsPerEpoch = n
	}
}

func (c *Config) normalize() error {
	if strings.TrimSpace(c.Model) != "" {
		model, err := NormalizeModel(c.Model)
		if err != nil {
			return err
		}
		c.Model = model
	}
	c.Dora.URL = strings.TrimRight(strings.TrimSpace(c.Dora.URL), "/")
	if strings.TrimSpace(c.Mongo.MongoURI) == "" {
		return fmt.Errorf("mongo.mongoURI is empty")
	}
	if strings.TrimSpace(c.Mongo.DBName) == "" {
		return fmt.Errorf("mongo.dbName is empty")
	}
	if c.SlotsPerEpoch == 0 {
		c.SlotsPerEpoch = 32
	}
	if c.SlotsPerEpoch < 1 {
		return fmt.Errorf("slots_per_epoch must be positive")
	}
	if c.TargetEpochs == 0 {
		c.TargetEpochs = 200
	}
	if c.TargetEpochs < 1 {
		return fmt.Errorf("target_epochs must be positive")
	}
	if c.TimeLoop <= 0 {
		c.TimeLoop = 10
	}
	if strings.TrimSpace(c.OutputDir) == "" {
		c.OutputDir = "output"
	}
	return nil
}

// RequireFetch checks the fields the collector needs. export only needs MongoDB.
func (c *Config) RequireFetch() error {
	if strings.TrimSpace(c.Model) == "" {
		return fmt.Errorf("fetch requires a model: set model in config.yaml or MODEL")
	}
	if c.Dora.URL == "" {
		return fmt.Errorf("fetch requires dora.url or DORA_URL")
	}
	return nil
}

// NormalizeModel maps a Kurtosis profile name onto the paper's weighting-model label.
// DF is the unmodified Lighthouse run, recorded as baseline.
func NormalizeModel(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "baseline", "df":
		return "baseline", nil
	case "srsw":
		return "srsw", nil
	case "lsw":
		return "lsw", nil
	case "desw":
		return "desw", nil
	default:
		return "", fmt.Errorf("model %q must be one of baseline, df, srsw, lsw, desw", raw)
	}
}
