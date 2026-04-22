package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Database   DatabaseConfig   `yaml:"database"`
	Feeds      FeedsConfig      `yaml:"feeds"`
	Expiry     ExpiryConfig     `yaml:"expiry"`
	Log        LogConfig        `yaml:"log"`
	XMPPServer XMPPServerConfig `yaml:"xmpp_server"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type FeedsConfig struct {
	PollInterval int    `yaml:"poll_interval"`
	UserAgent    string `yaml:"user_agent"`
}

type ExpiryConfig struct {
	SweepInterval  int `yaml:"sweep_interval"`
	HardDeleteAfter int `yaml:"hard_delete_after"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"` // path to log file; empty = silent. Ignored when -d is set.
}

type XMPPServerConfig struct {
	Domain string        `yaml:"domain"`
	TLS    XMPPTLSConfig `yaml:"tls"`
	C2S    C2SConfig     `yaml:"c2s"`
	C2STLS C2STLSConfig  `yaml:"c2s_tls"`
}

type XMPPTLSConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

// C2SConfig is the plain-TCP listener (port 5222 by default).
// When STARTTLS is true the server offers a TLS upgrade on this port.
type C2SConfig struct {
	Enabled  bool `yaml:"enabled"`
	Port     int  `yaml:"port"`
	STARTTLS bool `yaml:"starttls"`
}

// C2STLSConfig is the direct-TLS listener (port 5223 by default).
// TLS is always active on this port when enabled.
type C2STLSConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	cfg := &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "vectorcap.db",
		},
		Feeds: FeedsConfig{
			PollInterval: 60,
			UserAgent:    "VectorCore-EAG/1.0 (ops@example.com)",
		},
		Expiry: ExpiryConfig{
			SweepInterval:  300,
			HardDeleteAfter: 72,
		},
		Log: LogConfig{
			Level: "info",
		},
	}

	if err := yaml.NewDecoder(f).Decode(cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Expiry.SweepInterval <= 0 {
		cfg.Expiry.SweepInterval = 300
	}
	if cfg.Expiry.HardDeleteAfter <= 0 {
		cfg.Expiry.HardDeleteAfter = 72
	}
	if cfg.Feeds.PollInterval <= 0 {
		cfg.Feeds.PollInterval = 60
	}
	if cfg.XMPPServer.C2S.Port <= 0 {
		cfg.XMPPServer.C2S.Port = 5222
	}
	if cfg.XMPPServer.C2STLS.Port <= 0 {
		cfg.XMPPServer.C2STLS.Port = 5223
	}

	return cfg, nil
}
