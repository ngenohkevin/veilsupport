package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Port     string `mapstructure:"port"`
		Host     string `mapstructure:"host"`
		TorProxy string `mapstructure:"tor_proxy"`
	} `mapstructure:"server"`

	XMPP struct {
		Server   string `mapstructure:"server"`
		Admin    string `mapstructure:"admin"`
		Password string `mapstructure:"password"`
		Domain   string `mapstructure:"domain"`
	} `mapstructure:"xmpp"`

	Database struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"db_name"`
		SSLMode  string `mapstructure:"ssl_mode"`
	} `mapstructure:"database"`

	JWT struct {
		Secret string `mapstructure:"secret"`
		TTL    int    `mapstructure:"ttl"`
	} `mapstructure:"jwt"`

	Redis struct {
		URL string `mapstructure:"url"`
	} `mapstructure:"redis"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// Environment variable support
	viper.SetEnvPrefix("VEILSUPPORT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional if all required values are in env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("server.host", "localhost")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "veilsupport")
	viper.SetDefault("database.db_name", "veilsupport")
	viper.SetDefault("database.ssl_mode", "disable")

	// JWT defaults
	viper.SetDefault("jwt.ttl", 3600) // 1 hour in seconds

	// XMPP defaults
	viper.SetDefault("xmpp.domain", "localhost")
}