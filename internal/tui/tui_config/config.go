package tui_config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	BaseURL string `yaml:"base_url" toml:"base_url"`
	Token   string `yaml:"token" toml:"token"`
}

type LoadOptions struct {
	FlagBaseURL string
	FlagToken   string
}

func Load(opts LoadOptions) (Config, []string, error) {
	cfg := Config{BaseURL: "http://localhost:8080"}
	warnings := []string{}

	fileCfg, fileWarnings, err := loadFromFile()
	if err != nil {
		return Config{}, nil, err
	}
	warnings = append(warnings, fileWarnings...)
	merge(&cfg, fileCfg)

	if v := strings.TrimSpace(os.Getenv("MJR_BASE_URL")); v != "" {
		cfg.BaseURL = v
	}
	if v := strings.TrimSpace(os.Getenv("MJR_TOKEN")); v != "" {
		cfg.Token = v
	}

	if v := strings.TrimSpace(opts.FlagBaseURL); v != "" {
		cfg.BaseURL = v
	}
	if v := strings.TrimSpace(opts.FlagToken); v != "" {
		cfg.Token = v
	}

	return cfg, warnings, nil
}

func merge(dst *Config, src Config) {
	if v := strings.TrimSpace(src.BaseURL); v != "" {
		dst.BaseURL = v
	}
	if v := strings.TrimSpace(src.Token); v != "" {
		dst.Token = v
	}
}

func loadFromFile() (Config, []string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, nil, fmt.Errorf("get home dir: %w", err)
	}

	cfgDir := filepath.Join(home, ".config", "mjrwtf")
	yamlPath := filepath.Join(cfgDir, "config.yaml")
	tomlPath := filepath.Join(cfgDir, "config.toml")

	yamlExists := fileExists(yamlPath)
	tomlExists := fileExists(tomlPath)
	if !yamlExists && !tomlExists {
		return Config{}, nil, nil
	}

	warnings := []string{}
	path := ""
	format := ""
	if yamlExists {
		path = yamlPath
		format = "yaml"
		if tomlExists {
			warnings = append(warnings, "Both config.yaml and config.toml exist; using config.yaml")
		}
	} else {
		path = tomlPath
		format = "toml"
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	switch format {
	case "yaml":
		if err := yaml.Unmarshal(b, &cfg); err != nil {
			return Config{}, nil, fmt.Errorf("parse yaml config: %w", err)
		}
	case "toml":
		if err := toml.Unmarshal(b, &cfg); err != nil {
			return Config{}, nil, fmt.Errorf("parse toml config: %w", err)
		}
	default:
		panic("unreachable: format must be yaml or toml")
	}

	return cfg, warnings, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}
