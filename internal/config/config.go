package config

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	MinSessionSecretLen         = 32
	DefaultReceiveMax           = 512 << 20
	DefaultPath                 = "config.yaml"
	DefaultSiteName             = "Repo"
	DefaultBranch               = "main"
	DefaultMirrorPullInterval   = 5 * time.Minute
	DefaultMirrorWorkers        = 4
	DefaultMirrorCloneTimeout   = 15 * time.Minute
	DefaultMirrorFetchTimeout   = 10 * time.Minute
	DefaultMirrorPushTimeout    = 10 * time.Minute
	DefaultSoftDeletedRetention = 7 * 24 * time.Hour
)

type Config struct {
	HTTPAddr           string   `yaml:"http_addr"`
	DataDir            string   `yaml:"data_dir"`
	SiteName           string   `yaml:"site_name"`
	CookieSecure       bool     `yaml:"cookie_secure"`
	SessionSecret      string   `yaml:"session_secret"`
	MirrorSecretsKey   string   `yaml:"mirror_secrets_key"`
	AllowRegistration  bool     `yaml:"allow_registration"`
	TrustedProxies     []string `yaml:"trusted_proxies"`
	ReceiveMaxBytes    int64    `yaml:"receive_max_bytes"`
	MirrorPullInterval string   `yaml:"mirror_pull_interval"`
	TrustedProxyNets   []*net.IPNet
	MirrorPullEvery    time.Duration `yaml:"-"`

	MirrorWorkers      int    `yaml:"mirror_workers"`
	MirrorCloneTimeout string `yaml:"mirror_clone_timeout"`
	MirrorFetchTimeout string `yaml:"mirror_fetch_timeout"`
	MirrorPushTimeout  string `yaml:"mirror_push_timeout"`

	SoftDeletedRetention string `yaml:"soft_deleted_retention"`

	MirrorVerifyOpenPGPKeyring string `yaml:"mirror_verify_openpgp_keyring"`
	MirrorVerifyAllowedSigners string `yaml:"mirror_verify_allowed_signers"`

	MirrorCloneTimeoutDur   time.Duration `yaml:"-"`
	MirrorFetchTimeoutDur   time.Duration `yaml:"-"`
	MirrorPushTimeoutDur    time.Duration `yaml:"-"`
	SoftDeletedRetentionDur time.Duration `yaml:"-"`
}

func Default() Config {
	return Config{
		HTTPAddr:           ":8080",
		DataDir:            "data",
		SiteName:           DefaultSiteName,
		ReceiveMaxBytes:    DefaultReceiveMax,
		MirrorPullInterval: "5m",
		MirrorPullEvery:    DefaultMirrorPullInterval,

		MirrorWorkers:           DefaultMirrorWorkers,
		MirrorCloneTimeout:      "15m",
		MirrorFetchTimeout:      "10m",
		MirrorPushTimeout:       "10m",
		MirrorCloneTimeoutDur:   DefaultMirrorCloneTimeout,
		MirrorFetchTimeoutDur:   DefaultMirrorFetchTimeout,
		MirrorPushTimeoutDur:    DefaultMirrorPushTimeout,
		SoftDeletedRetention:    "168h",
		SoftDeletedRetentionDur: DefaultSoftDeletedRetention,
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

	if strings.TrimSpace(cfg.MirrorPullInterval) == "" {
		cfg.MirrorPullInterval = "5m"
	}

	every, err := time.ParseDuration(cfg.MirrorPullInterval)
	if err != nil {
		return Config{}, fmt.Errorf("mirror_pull_interval: %w", err)
	}

	cfg.MirrorPullEvery = every

	if err := cfg.normalizeMirror(); err != nil {
		return Config{}, err
	}

	nets, err := parseCIDRs(cfg.TrustedProxies)
	if err != nil {
		return Config{}, err
	}
	cfg.TrustedProxyNets = nets

	return cfg, nil
}

func (c *Config) normalizeMirror() error {
	if c.MirrorWorkers <= 0 {
		c.MirrorWorkers = DefaultMirrorWorkers
	}

	var err error
	if c.MirrorCloneTimeoutDur, err = parseDurationDefault("mirror_clone_timeout", c.MirrorCloneTimeout, DefaultMirrorCloneTimeout); err != nil {
		return err
	}

	if c.MirrorFetchTimeoutDur, err = parseDurationDefault("mirror_fetch_timeout", c.MirrorFetchTimeout, DefaultMirrorFetchTimeout); err != nil {
		return err
	}

	if c.MirrorPushTimeoutDur, err = parseDurationDefault("mirror_push_timeout", c.MirrorPushTimeout, DefaultMirrorPushTimeout); err != nil {
		return err
	}

	if c.SoftDeletedRetentionDur, err = parseDurationDefault("soft_deleted_retention", c.SoftDeletedRetention, DefaultSoftDeletedRetention); err != nil {
		return err
	}

	return nil
}

func parseDurationDefault(field, raw string, def time.Duration) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0, fmt.Errorf("%s: invalid duration %q", field, raw)
	}

	return d, nil
}

func (c Config) ValidateSecrets() error {
	if len(c.SessionSecret) < MinSessionSecretLen {
		return fmt.Errorf("session_secret required in config (min %d characters)", MinSessionSecretLen)
	}

	if k := strings.TrimSpace(c.MirrorSecretsKey); k != "" && len(k) < MinSessionSecretLen {
		return fmt.Errorf("mirror_secrets_key must be at least %d characters when set", MinSessionSecretLen)
	}

	return nil
}

func (c Config) MirrorSecretsMaterial() string {
	if k := strings.TrimSpace(c.MirrorSecretsKey); k != "" {
		return k
	}

	return c.SessionSecret
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
