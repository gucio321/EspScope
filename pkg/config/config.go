package config

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"github.com/kirsle/configdir"
)

const (
	configFile             = "config.json"
	AppID                  = "EspScope" // Used for config
	DefaultSamplesPerFrame = 1024
	DefaultFrequency       = 50000
	MinFrequency           = 0
	MaxFrequency           = 100000
)

type Config struct {
	DriftVoltage      float64
	LastIP            string
	VoltageMultiplyer float64
	AutoConnect       bool
	AutoStream        bool
	SamplesPerFrame   int32
	PosOnCrosshairs   bool
	LineThickness     int32
	LineColor         color.RGBA
	Frequency         int32
}

func NewConfig() *Config {
	return &Config{
		VoltageMultiplyer: 1,
		SamplesPerFrame:   DefaultSamplesPerFrame,
		PosOnCrosshairs:   true,
		LineThickness:     2,
		LineColor:         color.RGBA{R: 255, G: 0, B: 0, A: 255},
		Frequency:         DefaultFrequency,
	}
}

func ConfigPath() (string, error) {
	configPath := configdir.LocalConfig(AppID)
	if err := configdir.MakePath(configPath); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	return filepath.Join(configPath, configFile), nil
}

func (c *Config) Save() error {
	data, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("converting config to JSON: %w", err)
	}

	path, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("getting config path: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing config to %s: %w", path, err)
	}

	return nil
}

func (c *Config) Load() error {
	path, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("getting config file path: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config file from %s: %w", path, err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("decoding json configuration read from %s: %w", path, err)
	}

	return nil
}
