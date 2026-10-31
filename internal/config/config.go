package config

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	MinSessionSecretLen = 32
	DefaultReceiveMax   = 512 << 20
	DefaultPath         = "config.yaml"
	DefaultSiteName     = "Repo"
	DefaultBranch       = "main"
)

type Config struct {
	HTTPAddr          string   `yaml:"http_addr"`
	DataDir           string   `yaml:"data_dir"`
	SiteName          string   `yaml:"site_name"`
	CookieSecure      bool     `yaml:"cookie_secure"`
	SessionSecret     string   `yaml:"session_secret"`
	AllowRegistration bool     `yaml:"allow_registration"`
	TrustedProxies    []string `yaml:"trusted_proxies"`
	ReceiveMaxBytes   int64    `yaml:"receive_max_bytes"`
	TrustedProxyNets  []*net.IPNet
}

func Default() Config {
	return Config{
		HTTPAddr:        ":8080",
		DataDir:         "data",
		SiteName:        DefaultSiteName,
		ReceiveMaxBytes: DefaultReceiveMax,
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := Default()
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.HTTPAddr == "" {
		cfg.HTTPAddr = ":8080"
	}

	if cfg.DataDir == "" {
		cfg.DataDir = "data"
	}

	cfg.SiteName = strings.TrimSpace(cfg.SiteName)
	if cfg.SiteName == "" {
		cfg.SiteName = DefaultSiteName
	}

	if cfg.ReceiveMaxBytes <= 0 {
		cfg.ReceiveMaxBytes = DefaultReceiveMax
	}

	nets, err := parseCIDRs(cfg.TrustedProxies)
	if err != nil {
		return Config{}, err
	}
	cfg.TrustedProxyNets = nets

	return cfg, nil
}

func (c Config) ValidateSecrets() error {
	if len(c.SessionSecret) < MinSessionSecretLen {
		return fmt.Errorf("session_secret required in config (min %d characters)", MinSessionSecretLen)
	}

	return nil
}

func parseCIDRs(cidrs []string) ([]*net.IPNet, error) {
	if len(cidrs) == 0 {
		return nil, nil
	}

	out := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		if !strings.Contains(raw, "/") {
			ip := net.ParseIP(raw)
			if ip == nil {
				return nil, fmt.Errorf("trusted_proxies: invalid IP %q", raw)
			}

			if ip.To4() != nil {
				raw = ip.String() + "/32"
			} else {
				raw = ip.String() + "/128"
			}
		}

		_, network, err := net.ParseCIDR(raw)
		if err != nil {
			return nil, fmt.Errorf("trusted_proxies: %w", err)
		}

		out = append(out, network)
	}

	return out, nil
}
