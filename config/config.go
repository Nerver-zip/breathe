package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nerver-zip/breathing-tui/internal/theme"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Rounds        int           `mapstructure:"rounds" yaml:"rounds"`
	Breathing     time.Duration `mapstructure:"breathing" yaml:"breathing"`
	Recovery      time.Duration `mapstructure:"recovery" yaml:"recovery"`
	AutoNextRound bool          `mapstructure:"auto_next_round" yaml:"auto_next_round"`
	Theme         string        `mapstructure:"theme" yaml:"theme"`
	Notifications bool          `mapstructure:"notifications" yaml:"notifications"`
	Bell          bool          `mapstructure:"bell" yaml:"bell"`
	Font          string        `mapstructure:"font" yaml:"font"`
	Mode          string        `mapstructure:"mode" yaml:"mode"`
	Breaths       int           `mapstructure:"breaths" yaml:"breaths"`
}

func Defaults() Config {
	return Config{
		Rounds:        3,
		Breathing:     3 * time.Minute,
		Recovery:      30 * time.Second,
		AutoNextRound: false,
		Theme:         "default",
		Notifications: true,
		Bell:          true,
		Font:          "ansiShadow",
		Mode:          "timed",
		Breaths:       30,
	}
}

func Dir() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "breath"), nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "breath"), nil
}

func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (Config, error) {
	cfg := Defaults()
	path, err := Path()
	if err != nil {
		return cfg, err
	}

	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.SetDefault("rounds", cfg.Rounds)
	v.SetDefault("breathing", cfg.Breathing.String())
	v.SetDefault("recovery", cfg.Recovery.String())
	v.SetDefault("auto_next_round", cfg.AutoNextRound)
	v.SetDefault("theme", cfg.Theme)
	v.SetDefault("notifications", cfg.Notifications)
	v.SetDefault("bell", cfg.Bell)
	v.SetDefault("font", cfg.Font)
	v.SetDefault("mode", cfg.Mode)
	v.SetDefault("breaths", cfg.Breaths)

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return cfg, err
		}
		if err := Save(cfg); err != nil {
			return cfg, err
		}
	}

	cfg.Rounds = v.GetInt("rounds")
	cfg.AutoNextRound = v.GetBool("auto_next_round")
	cfg.Theme = v.GetString("theme")
	cfg.Notifications = v.GetBool("notifications")
	cfg.Bell = v.GetBool("bell")
	if f := v.GetString("font"); f != "" {
		cfg.Font = f
	}
	if m := strings.ToLower(v.GetString("mode")); m == "counted" {
		cfg.Mode = "counted"
	} else {
		cfg.Mode = "timed"
	}
	if b := v.GetInt("breaths"); b >= 1 {
		cfg.Breaths = b
	} else {
		cfg.Breaths = 30
	}

	if d, err := time.ParseDuration(v.GetString("breathing")); err == nil {
		cfg.Breathing = d
	} else {
		return cfg, fmt.Errorf("invalid breathing duration: %w", err)
	}
	if d, err := time.ParseDuration(v.GetString("recovery")); err == nil {
		cfg.Recovery = d
	} else {
		return cfg, fmt.Errorf("invalid recovery duration: %w", err)
	}

	if cfg.Rounds < 1 {
		return cfg, fmt.Errorf("rounds must be >= 1")
	}
	if cfg.Breathing <= 0 || cfg.Recovery <= 0 {
		return cfg, fmt.Errorf("durations must be > 0")
	}
	if _, err := theme.Get(cfg.Theme); err != nil {
		cfg.Theme = "default"
	}
	return cfg, nil
}

type yamlConfig struct {
	Rounds        int    `yaml:"rounds"`
	Breathing     string `yaml:"breathing"`
	Recovery      string `yaml:"recovery"`
	AutoNextRound bool   `yaml:"auto_next_round"`
	Theme         string `yaml:"theme"`
	Notifications bool   `yaml:"notifications"`
	Bell          bool   `yaml:"bell"`
	Font          string `yaml:"font"`
	Mode          string `yaml:"mode"`
	Breaths       int    `yaml:"breaths"`
}

func Save(cfg Config) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	y := yamlConfig{
		Rounds:        cfg.Rounds,
		Breathing:     cfg.Breathing.String(),
		Recovery:      cfg.Recovery.String(),
		AutoNextRound: cfg.AutoNextRound,
		Theme:         cfg.Theme,
		Notifications: cfg.Notifications,
		Bell:          cfg.Bell,
		Font:          cfg.Font,
		Mode:          cfg.Mode,
		Breaths:       cfg.Breaths,
	}

	data, err := yaml.Marshal(y)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	// Atomic file write using a temporary file in the same directory
	tmpPath := fmt.Sprintf("%s.tmp.%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}

var SupportedKeys = []string{
	"rounds",
	"breathing",
	"recovery",
	"auto_next_round",
	"theme",
	"notifications",
	"bell",
	"font",
	"mode",
	"breaths",
}

func Set(key, value string) (Config, error) {
	cfg, err := Load()
	if err != nil {
		return cfg, err
	}

	key = strings.ToLower(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "_")
	value = strings.TrimSpace(value)

	switch key {
	case "rounds":
		val, err := strconv.Atoi(value)
		if err != nil || val < 1 {
			return cfg, fmt.Errorf("rounds must be an integer >= 1 (got %q)", value)
		}
		cfg.Rounds = val
	case "breathing":
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return cfg, fmt.Errorf("breathing must be a positive duration like '3m' or '2m30s' (got %q)", value)
		}
		cfg.Breathing = d
	case "recovery":
		d, err := time.ParseDuration(value)
		if err != nil || d <= 0 {
			return cfg, fmt.Errorf("recovery must be a positive duration like '30s' or '1m' (got %q)", value)
		}
		cfg.Recovery = d
	case "auto_next_round":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return cfg, fmt.Errorf("auto_next_round must be true or false (got %q)", value)
		}
		cfg.AutoNextRound = b
	case "theme":
		th, err := theme.Get(value)
		if err != nil {
			return cfg, err
		}
		cfg.Theme = th.Name
	case "notifications":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return cfg, fmt.Errorf("notifications must be true or false (got %q)", value)
		}
		cfg.Notifications = b
	case "bell":
		b, err := strconv.ParseBool(value)
		if err != nil {
			return cfg, fmt.Errorf("bell must be true or false (got %q)", value)
		}
		cfg.Bell = b
	case "font":
		switch strings.ToLower(value) {
		case "ansishadow", "ansi_shadow", "ansi-shadow":
			cfg.Font = "ansiShadow"
		case "mono12", "mono_12", "mono-12":
			cfg.Font = "mono12"
		case "ansi":
			cfg.Font = "ansi"
		case "rebel":
			cfg.Font = "rebel"
		default:
			return cfg, fmt.Errorf("unknown font %q. Available fonts: ansiShadow, mono12, ansi, rebel", value)
		}
	case "mode":
		switch strings.ToLower(value) {
		case "timed", "countdown":
			cfg.Mode = "timed"
		case "counted", "counter", "breaths":
			cfg.Mode = "counted"
		default:
			return cfg, fmt.Errorf("mode must be 'timed' or 'counted' (got %q)", value)
		}
	case "breaths":
		val, err := strconv.Atoi(value)
		if err != nil || val < 1 {
			return cfg, fmt.Errorf("breaths must be an integer >= 1 (got %q)", value)
		}
		cfg.Breaths = val
	default:
		return cfg, fmt.Errorf("unknown configuration key %q. Supported keys: %s", key, strings.Join(SupportedKeys, ", "))
	}

	if err := Save(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
