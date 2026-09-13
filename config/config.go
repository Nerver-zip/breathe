package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Rounds        int           `mapstructure:"rounds" yaml:"rounds"`
	Breathing     time.Duration `mapstructure:"breathing" yaml:"breathing"`
	Recovery      time.Duration `mapstructure:"recovery" yaml:"recovery"`
	AutoNextRound bool          `mapstructure:"auto_next_round" yaml:"auto_next_round"`
	Theme         string        `mapstructure:"theme" yaml:"theme"`
	Notifications bool          `mapstructure:"notifications" yaml:"notifications"`
}

func Defaults() Config {
	return Config{
		Rounds:        3,
		Breathing:     3 * time.Minute,
		Recovery:      30 * time.Second,
		AutoNextRound: false,
		Theme:         "default",
		Notifications: true,
	}
}

func Dir() (string, error) {
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

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return cfg, err
		}
		if err := writeDefaults(v, path); err != nil {
			return cfg, err
		}
	}

	cfg.Rounds = v.GetInt("rounds")
	cfg.AutoNextRound = v.GetBool("auto_next_round")
	cfg.Theme = v.GetString("theme")
	cfg.Notifications = v.GetBool("notifications")

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
	return cfg, nil
}

func writeDefaults(v *viper.Viper, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return v.WriteConfigAs(path)
}
