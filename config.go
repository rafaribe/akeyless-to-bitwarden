package main

import (
	"flag"
	"log"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Akeyless struct {
		URL       string `mapstructure:"url"`
		AccessID  string `mapstructure:"access_id"`
		AccessKey string `mapstructure:"access_key"`
	} `mapstructure:"akeyless"`
	Bitwarden struct {
		APIURL      string `mapstructure:"api_url"`
		IdentityURL string `mapstructure:"identity_url"`
		AccessToken string `mapstructure:"access_token"`
		OrgID       string `mapstructure:"org_id"`
		ProjectID   string `mapstructure:"project_id"`
	} `mapstructure:"bitwarden"`
}

func LoadConfig() *Config {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	if *configPath != "" {
		viper.SetConfigFile(*configPath)
	} else if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		viper.SetConfigFile(envPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	return &cfg
}
