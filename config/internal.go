package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"os"
	"time"
)

type Config struct {
	Env         string `yaml:"env" env-default:"local"`
	StoragePath string `yaml:"storage_path" emv-default:"storage/mySql.go"`
	HTTPServer  `yaml:"http_server"`
	JiraConfig  `yaml:"jira_config"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env-default:":8000"`
	Timeout     time.Duration `yaml:"timeout" env-default:"5s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"5s"`
}

type JiraConfig struct {
	JiraURL  string `yaml:"jira_url" env:"JIRA_URL"`
	Username string `yaml:"username" env:"JIRA_USER"`
	Password string `yaml:"token" env:"JIRA_TOKEN"`
}

func MustLoadConfig() *Config {
	configPath := os.Getenv("CONFIG_PATH")

	if configPath == "" {
		panic("CONFIG_PATH environment variable not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		panic("CONFIG_PATH does not exist")
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		panic("failed to read config: " + err.Error())
	}

	return &config
}
