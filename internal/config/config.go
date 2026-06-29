package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Workers int    `yaml:"workers"`
}

type CPUConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Threads     int    `yaml:"threads"`
	AVX512      string `yaml:"avx512"`
	AMX         string `yaml:"amx"`
	MemoryLimit string `yaml:"memory_limit"`
}

type GPUConfig struct {
	Enabled       string `yaml:"enabled"`
	Devices       []int  `yaml:"devices"`
	VRAMLimit     string `yaml:"vram_limit"`
	FallbackToCPU bool   `yaml:"fallback_to_cpu"`
}

type HardwareConfig struct {
	CPU CPUConfig `yaml:"cpu"`
	GPU GPUConfig `yaml:"gpu"`
}

type ModelSourceConfig struct {
	Dir     string   `yaml:"dir"`
	Sources []string `yaml:"sources"`
}

type ModelEntryConfig struct {
	URL        string `yaml:"url"`
	SHA256     string `yaml:"sha256"`
	VoicesURL  string `yaml:"voices_url,omitempty"`
}

type ModelsConfig struct {
	Dir          string                       `yaml:"dir"`
	RegistryURL  string                       `yaml:"registry_url"`
	Sources      []string                     `yaml:"sources"`
	Entries      map[string]ModelEntryConfig  `yaml:"entries,omitempty"`
}

type AuthConfig struct {
	Enabled bool `yaml:"enabled"`
}

type RateLimitConfig struct {
	Enabled bool `yaml:"enabled"`
}

type CORSConfig struct {
	Enabled bool     `yaml:"enabled"`
	Origins []string `yaml:"origins"`
}

type APIConfig struct {
	Auth      AuthConfig      `yaml:"auth"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	CORS      CORSConfig      `yaml:"cors"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
	Output string `yaml:"output"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Hardware HardwareConfig `yaml:"hardware"`
	Models   ModelsConfig   `yaml:"models"`
	API      APIConfig      `yaml:"api"`
	Logging  LoggingConfig  `yaml:"logging"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    "0.0.0.0",
			Port:    11435,
			Workers: 4,
		},
		Hardware: HardwareConfig{
			CPU: CPUConfig{
				Enabled:     true,
				Threads:     0,
				AVX512:      "auto",
				AMX:         "auto",
				MemoryLimit: "8GB",
			},
			GPU: GPUConfig{
				Enabled:       "auto",
				Devices:       []int{0},
				VRAMLimit:     "12GB",
				FallbackToCPU: true,
			},
		},
		Models: ModelsConfig{
			Dir:         "/var/aurora/models",
			RegistryURL: "http://localhost:8000",
			Sources:     []string{"https://cdn.aurora.ai/models/"},
		},
		API: APIConfig{
			Auth: AuthConfig{Enabled: false},
			RateLimit: RateLimitConfig{Enabled: false},
			CORS: CORSConfig{
				Enabled: true,
				Origins: []string{"*"},
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return cfg, nil
}
