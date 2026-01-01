package tui_config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaskToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: "<empty>"},
		{name: "spaces", in: "   ", want: "<empty>"},
		{name: "len3", in: "abc", want: "****"},
		{name: "len4", in: "abcd", want: "****"},
		{name: "len5", in: "abcde", want: "ab*de"},
		{name: "len6", in: "abcdef", want: "ab**ef"},
		{name: "trim", in: "  abcdef  ", want: "ab**ef"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := MaskToken(tc.in); got != tc.want {
				t.Fatalf("MaskToken(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("MJR_BASE_URL", "")
	t.Setenv("MJR_TOKEN", "")
	t.Setenv("HOME", t.TempDir())

	cfg, warnings, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", warnings)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:8080")
	}
	if cfg.Token != "" {
		t.Fatalf("Token = %q, want empty", cfg.Token)
	}
}

func TestLoad_FileYAMLPreferredWithWarning(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MJR_BASE_URL", "")
	t.Setenv("MJR_TOKEN", "")

	cfgDir := filepath.Join(home, ".config", "mjrwtf")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	yamlPath := filepath.Join(cfgDir, "config.yaml")
	tomlPath := filepath.Join(cfgDir, "config.toml")

	if err := os.WriteFile(yamlPath, []byte("base_url: http://yaml\ntoken: yamltoken\n"), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}
	if err := os.WriteFile(tomlPath, []byte("base_url = \"http://toml\"\ntoken = \"tomltoken\"\n"), 0o600); err != nil {
		t.Fatalf("write toml: %v", err)
	}

	cfg, warnings, err := Load(LoadOptions{})
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.BaseURL != "http://yaml" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "http://yaml")
	}
	if cfg.Token != "yamltoken" {
		t.Fatalf("Token = %q, want %q", cfg.Token, "yamltoken")
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "using config.yaml") {
		t.Fatalf("expected yaml preference warning, got %v", warnings)
	}
}

func TestLoad_Precedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgDir := filepath.Join(home, ".config", "mjrwtf")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte("base_url: http://file\ntoken: filetoken\n"), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	t.Run("file", func(t *testing.T) {
		t.Setenv("MJR_BASE_URL", "")
		t.Setenv("MJR_TOKEN", "")
		cfg, _, err := Load(LoadOptions{})
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.BaseURL != "http://file" || cfg.Token != "filetoken" {
			t.Fatalf("got %+v", cfg)
		}
	})

	t.Run("env_over_file", func(t *testing.T) {
		t.Setenv("MJR_BASE_URL", "http://env")
		t.Setenv("MJR_TOKEN", "envtoken")
		cfg, _, err := Load(LoadOptions{})
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.BaseURL != "http://env" || cfg.Token != "envtoken" {
			t.Fatalf("got %+v", cfg)
		}
	})

	t.Run("flags_over_env", func(t *testing.T) {
		t.Setenv("MJR_BASE_URL", "http://env")
		t.Setenv("MJR_TOKEN", "envtoken")
		cfg, _, err := Load(LoadOptions{FlagBaseURL: "http://flag", FlagToken: "flagtoken"})
		if err != nil {
			t.Fatalf("Load() error: %v", err)
		}
		if cfg.BaseURL != "http://flag" || cfg.Token != "flagtoken" {
			t.Fatalf("got %+v", cfg)
		}
	})
}
