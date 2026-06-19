package config

import (
	"bytes"
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port        string        `yaml:"port"`
		UpstreamURL string        `yaml:"upstream_url"`
		CacheTTL    time.Duration `yaml:"cache_ttl"`
	} `yaml:"server"`
	Redis struct {
		Addr        string        `yaml:"addr"`
		Password    string        `yaml:"password"`
		User        string        `yaml:"user"`
		DB          int           `yaml:"db"`
		MaxRetries  int           `yaml:"max_retries"`
		DialTimeout time.Duration `yaml:"dial_timeout"`
		Timeout     time.Duration `yaml:"timeout"`
	} `yaml:"redis"`
}

func Init() *Config {
	cfgRaw, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	var buffer bytes.Buffer
	buffer.Write(cfgRaw)
	var config Config
	if err = yaml.Unmarshal(buffer.Bytes(), &config); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	return &config
}
