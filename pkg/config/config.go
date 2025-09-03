package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App      App      `yaml:"app"`
	Database Database `yaml:"database"`
	Allows   Allows   `yaml:"allows"`
}

type App struct {
	Name string `yaml:"name"`
	Port string `yaml:"port"`
	Host string `yaml:"host"`
}

type Database struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
	Name string `yaml:"name"`
}

type Allows struct {
	Methods []string `yaml:"methods"`
	Origins []string `yaml:"origins"`
	Headers []string `yaml:"headers"`
}

func InitConfig() *Config {
	var configs Config
	file_name, _ := filepath.Abs("./config.yaml")
	yaml_file, _ := os.ReadFile(file_name)
	yaml.Unmarshal(yaml_file, &configs)

	// Override with environment variables if they exist (for Docker)
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		configs.Database.Host = dbHost
	}
	if dbPort := os.Getenv("DB_PORT"); dbPort != "" {
		configs.Database.Port = dbPort
	}
	if dbUser := os.Getenv("DB_USER"); dbUser != "" {
		configs.Database.User = dbUser
	}
	if dbPassword := os.Getenv("DB_PASSWORD"); dbPassword != "" {
		configs.Database.Pass = dbPassword
	}
	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		configs.Database.Name = dbName
	}

	// Override app configuration with environment variables
	if appHost := os.Getenv("APP_HOST"); appHost != "" {
		configs.App.Host = appHost
	}
	if appPort := os.Getenv("APP_PORT"); appPort != "" {
		configs.App.Port = appPort
	}
	if appName := os.Getenv("APP_NAME"); appName != "" {
		configs.App.Name = appName
	}

	return &configs
}
