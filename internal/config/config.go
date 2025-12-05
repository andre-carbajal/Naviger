package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const (
	defaultConfigName   = "config.json"
	defaultServersDir   = "servers"
	defaultBackupsDir   = "backups"
	defaultRuntimesDir  = "runtimes"
	defaultDatabaseFile = "manager.db"
	defaultPort         = 8080
)

type Config struct {
	ServersPath  string `json:"servers_path"`
	BackupsPath  string `json:"backups_path"`
	RuntimesPath string `json:"runtimes_path"`
	DatabasePath string `json:"database_path"`
	Port         int    `json:"port"`
}

func LoadConfig(configDir string) (*Config, error) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, defaultConfigName)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath, configDir)
	}

	file, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(file, &cfg); err != nil {
		return nil, err
	}

	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}

	return &cfg, nil
}

func createDefaultConfig(configPath, configDir string) (*Config, error) {
	cfg := Config{
		ServersPath:  filepath.Join(configDir, defaultServersDir),
		BackupsPath:  filepath.Join(configDir, defaultBackupsDir),
		RuntimesPath: filepath.Join(configDir, defaultRuntimesDir),
		DatabasePath: filepath.Join(configDir, defaultDatabaseFile),
		Port:         defaultPort,
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, err
	}

	return &cfg, nil
}
