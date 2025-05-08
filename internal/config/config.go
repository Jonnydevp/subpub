package config

import (
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

type Config struct {
	GRPC struct {
		Addr string `yaml:"addr" envconfig:"GRPC_ADDR"`
	} `yaml:"grpc"`
	Log struct {
		Level string `yaml:"level" envconfig:"LOG_LEVEL"`
	} `yaml:"log"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" envconfig:"SHUTDOWN_TIMEOUT"`
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		GRPC: struct {
			Addr string `yaml:"addr" envconfig:"GRPC_ADDR"`
		}{
			Addr: ":50051",
		},
		Log: struct {
			Level string `yaml:"level" envconfig:"LOG_LEVEL"`
		}{
			Level: "info",
		},
		ShutdownTimeout: 10 * time.Second,
	}

	if path != "" {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
			return nil, err
		}
	}

	if err := envconfig.Process("", cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
