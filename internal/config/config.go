package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Database DatabaseConfig `mapstructure:"database"`
	Scraper  ScraperConfig  `mapstructure:"scraper"`
	AI       AIConfig       `mapstructure:"ai"`
	Server   ServerConfig   `mapstructure:"server"`
}

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type ScraperConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Interval int    `mapstructure:"interval"`
	DLTURL   string `mapstructure:"dlt_url"`
	SSQURL   string `mapstructure:"ssq_url"`
}

type AIConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	APIURL  string `mapstructure:"api_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
	Timeout int    `mapstructure:"timeout"`
}

type ServerConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Host    string `mapstructure:"host"`
}

var globalConfig *Config

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &cfg
	return &cfg, nil
}

func Get() *Config {
	return globalConfig
}
