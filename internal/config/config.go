package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config is the root configuration structure loaded from config.toml.
type Config struct {
	Tuya        TuyaConfig        `toml:"tuya"`
	WoL         WoLConfig         `toml:"wol"`
	Workstation WorkstationConfig `toml:"workstation"`
}

// TuyaConfig holds the Tuya IoT Cloud developer credentials and device binding.
type TuyaConfig struct {
	// ClientID is the Access ID from iot.tuya.com → Cloud Project → Overview.
	ClientID string `toml:"client_id"`
	// ClientSecret is the Access Secret from the same page.
	ClientSecret string `toml:"client_secret"`
	// DeviceID is the virtual device ID created in the Tuya platform.
	DeviceID string `toml:"device_id"`
	// Region selects the data-center: "eu", "us", "cn", or "in".
	Region string `toml:"region"`
	// SwitchDP is the data-point code that represents the power switch (default "switch_1").
	SwitchDP string `toml:"switch_dp"`
	// DebugEvents logs every raw MQTT event received from Tuya (useful during setup).
	DebugEvents bool `toml:"debug_events"`
}

// WoLConfig describes the target machine to wake up.
type WoLConfig struct {
	// MACAddress is the MAC of the target machine in any standard notation.
	MACAddress string `toml:"mac_address"`
	// BroadcastIP is the directed/limited broadcast address (e.g. "192.168.1.255" or "255.255.255.255").
	BroadcastIP string `toml:"broadcast_ip"`
	// Port is the UDP destination port for the magic packet (usually 7 or 9).
	Port int `toml:"port"`
	// Repeat controls how many times the packet is sent (default 3).
	Repeat int `toml:"repeat"`
}

// WorkstationConfig enables the periodic system-stats reporter.
// Enable this only on the machine running as workstation; leave it false on the Pi.
type WorkstationConfig struct {
	// Enabled activates CPU / RAM / Disk / VRAM reporting.
	Enabled bool `toml:"enabled"`
	// StatsInterval is the reporting interval in seconds (default 60).
	StatsInterval int `toml:"stats_interval"`
}

// Load reads a TOML file at path and returns a validated Config.
func Load(path string) (*Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	cfg := defaults()
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Tuya: TuyaConfig{
			Region:   "eu",
			SwitchDP: "switch_1",
		},
		WoL: WoLConfig{
			BroadcastIP: "255.255.255.255",
			Port:        9,
			Repeat:      3,
		},
		Workstation: WorkstationConfig{
			StatsInterval: 60,
		},
	}
}

func (c *Config) validate() error {
	if c.Tuya.ClientID == "" {
		return fmt.Errorf("tuya.client_id is required")
	}
	if c.Tuya.ClientSecret == "" {
		return fmt.Errorf("tuya.client_secret is required")
	}
	if c.Tuya.DeviceID == "" {
		return fmt.Errorf("tuya.device_id is required")
	}
	if c.WoL.MACAddress == "" {
		return fmt.Errorf("wol.mac_address is required")
	}
	switch c.Tuya.Region {
	case "eu", "us", "cn", "in":
	default:
		return fmt.Errorf("tuya.region must be one of: eu, us, cn, in")
	}
	return nil
}
