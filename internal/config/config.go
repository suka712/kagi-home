// Package config loads and saves the vault's config.yaml.
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type OllamaConfig struct {
	Host        string  `yaml:"host"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
}

type Config struct {
	Ollama OllamaConfig `yaml:"ollama"`
}

func Default() Config {
	return Config{
		Ollama: OllamaConfig{
			Host:        "http://localhost:11434",
			Model:       "gemma4:e2b",
			Temperature: 0.3,
		},
	}
}

func path(vaultDir string) string {
	return filepath.Join(vaultDir, "config.yaml")
}

// Load reads config.yaml from the vault, falling back to defaults for any
// unset fields (or the whole file if it doesn't exist yet).
func Load(vaultDir string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path(vaultDir))
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func Save(vaultDir string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path(vaultDir), data, 0o644)
}
