package gui

type Config struct {
	DriftVoltage float64
}

func NewConfig() *Config {
	return &Config{}
}
