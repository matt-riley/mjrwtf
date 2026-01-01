package tui_config

import (
	"errors"
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

	Source string `yaml:"-" toml:"-"`
}

type LoadOptions struct {
	FlagBaseURL string
	FlagToken   string
}

func Load(opts LoadOptions) (Config, []string, error) {
	cfg := Config{BaseURL: "http://localhost:8080", Source: "default"}
	warnings := []string{}

	fileCfg, fileWarnings, err := loadFromFile()
	if err != nil {
		return Config{}, nil, err
	}
	warnings = append(warnings, fileWarnings...)
	merge(&cfg, fileCfg, "config")

	if v := strings.TrimSpace(os.Getenv("MJR_BASE_URL")); v != "" {
		cfg.BaseURL = v
		cfg.Source = "env"
	}
	if v := strings.TrimSpace(os.Getenv("MJR_TOKEN")); v != "" {
		cfg.Token = v
		cfg.Source = "env"
	}

	if strings.TrimSpace(opts.FlagBaseURL) != "" {
		cfg.BaseURL = strings.TrimSpace(opts.FlagBaseURL)
		cfg.Source = "flag"
	}
	if strings.TrimSpace(opts.FlagToken) != "" {
		cfg.Token = strings.TrimSpace(opts.FlagToken)
		cfg.Source = "flag"
	}

	return cfg, warnings, nil
}

func merge(dst *Config, src Config, source string) {
	if strings.TrimSpace(src.BaseURL) != "" {
		dst.BaseURL = strings.TrimSpace(src.BaseURL)
		dst.Source = source
	}
	if strings.TrimSpace(src.Token) != "" {
		dst.Token = strings.TrimSpace(src.Token)
		dst.Source = source
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
		return Config{}, nil, errors.New("unknown config format")
	}

	cfg.Source = "config_file"
	return cfg, warnings, nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !st.IsDir()
}
