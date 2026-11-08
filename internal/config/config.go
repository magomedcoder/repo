package config

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"magomedcoder.ru/repo/pkg/netutil"
)

const (
	MinSessionSecretLen         = 32
	MinSecretLen                = MinSessionSecretLen
	DefaultReceiveMax           = 256 << 20 // 256 MiB
	DefaultPath                 = "config.yaml"
	DefaultSiteName             = "Repo"
	DefaultBranch               = "main"
	DefaultMirrorPullInterval   = 5 * time.Minute
	DefaultMirrorWorkers        = 4
	DefaultMirrorCloneTimeout   = 15 * time.Minute
	DefaultMirrorFetchTimeout   = 10 * time.Minute
	DefaultMirrorPushTimeout    = 10 * time.Minute
	DefaultSoftDeletedRetention = 7 * 24 * time.Hour
	DefaultMirrorMaxBytes       = 5 << 30 // 5 GiB
	DefaultSessionAbsolute      = 30 * 24 * time.Hour
	DefaultSessionIdle          = 14 * 24 * time.Hour
	DefaultHTTPAddr             = "127.0.0.1:8080"
)

type Config struct {
	HTTP             HTTPConfig    `yaml:"http"`
	Server           ServerConfig  `yaml:"server"`
	Secrets          SecretsConfig `yaml:"secrets"`
	Auth             AuthConfig    `yaml:"auth"`
	Mirror           MirrorConfig  `yaml:"mirror"`
	TrustedProxyNets []*net.IPNet  `yaml:"-"`
}

type HTTPConfig struct {
	Addr            string   `yaml:"addr"`
	CookieSecure    bool     `yaml:"cookie_secure"`
	TrustedProxies  []string `yaml:"trusted_proxies"`
	ReceiveMaxBytes int64    `yaml:"receive_max_bytes"`
}

type ServerConfig struct {
	DataDir                 string        `yaml:"data_dir"`
	SiteName                string        `yaml:"site_name"`
	SoftDeletedRetention    string        `yaml:"soft_deleted_retention"`
	SoftDeletedRetentionDur time.Duration `yaml:"-"`
}

type SecretsConfig struct {
	SessionSecret    string `yaml:"session_secret"`
	TokenPepper      string `yaml:"token_pepper"`
	CSRFSecret       string `yaml:"csrf_secret"`
	MirrorSecretsKey string `yaml:"mirror_secrets_key"`
}

type AuthConfig struct {
	AllowRegistration     bool   `yaml:"allow_registration"`
	AllowPublicContribute bool   `yaml:"allow_public_contribute"`
	RequireAuthForRead    bool   `yaml:"require_auth_for_read"`
	GitPasswordAuth       bool   `yaml:"git_password_auth"`
	SessionAbsolute       string `yaml:"session_absolute"`
	SessionIdle           string `yaml:"session_idle"`

	SessionAbsoluteDur time.Duration `yaml:"-"`
	SessionIdleDur     time.Duration `yaml:"-"`
}

type MirrorConfig struct {
	PullInterval    string             `yaml:"pull_interval"`
	Workers         int                `yaml:"workers"`
	CloneTimeout    string             `yaml:"clone_timeout"`
	FetchTimeout    string             `yaml:"fetch_timeout"`
	PushTimeout     string             `yaml:"push_timeout"`
	DefaultMaxBytes int64              `yaml:"default_max_bytes"`
	Verify          MirrorVerifyConfig `yaml:"verify"`

	PullEvery       time.Duration `yaml:"-"`
	CloneTimeoutDur time.Duration `yaml:"-"`
	FetchTimeoutDur time.Duration `yaml:"-"`
	PushTimeoutDur  time.Duration `yaml:"-"`
}

type MirrorVerifyConfig struct {
	OpenPGPKeyring string `yaml:"openpgp_keyring"`
	AllowedSigners string `yaml:"allowed_signers"`
}

func Default() Config {
	return Config{
		HTTP: HTTPConfig{
			Addr:            DefaultHTTPAddr,
			ReceiveMaxBytes: DefaultReceiveMax,
		},
		Server: ServerConfig{
			DataDir:                 "data",
			SiteName:                DefaultSiteName,
			SoftDeletedRetention:    "168h",
			SoftDeletedRetentionDur: DefaultSoftDeletedRetention,
		},
		Auth: AuthConfig{
			GitPasswordAuth:    true,
			SessionAbsolute:    "720h",
			SessionIdle:        "336h",
			SessionAbsoluteDur: DefaultSessionAbsolute,
			SessionIdleDur:     DefaultSessionIdle,
		},
		Mirror: MirrorConfig{
			PullInterval:    "5m",
			PullEvery:       DefaultMirrorPullInterval,
			Workers:         DefaultMirrorWorkers,
			CloneTimeout:    "15m",
			FetchTimeout:    "10m",
			PushTimeout:     "10m",
			DefaultMaxBytes: DefaultMirrorMaxBytes,
			CloneTimeoutDur: DefaultMirrorCloneTimeout,
			FetchTimeoutDur: DefaultMirrorFetchTimeout,
			PushTimeoutDur:  DefaultMirrorPushTimeout,
		},
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		path = DefaultPath
	}

	info, err := os.Stat(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	if mode := info.Mode().Perm(); mode&0o044 != 0 {
		return Config{}, fmt.Errorf("config %s has mode %04o: it holds secrets and must not be group/world-readable; run chmod 600 %s", path, mode, path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	cfg := Default()
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = DefaultHTTPAddr
	}

	if cfg.Server.DataDir == "" {
		cfg.Server.DataDir = "data"
	}

	cfg.Server.SiteName = strings.TrimSpace(cfg.Server.SiteName)
	if cfg.Server.SiteName == "" {
		cfg.Server.SiteName = DefaultSiteName
	}

	if cfg.HTTP.ReceiveMaxBytes <= 0 {
		cfg.HTTP.ReceiveMaxBytes = DefaultReceiveMax
	}

	if strings.TrimSpace(cfg.Mirror.PullInterval) == "" {
		cfg.Mirror.PullInterval = "5m"
	}

	every, err := time.ParseDuration(cfg.Mirror.PullInterval)
	if err != nil {
		return Config{}, fmt.Errorf("mirror.pull_interval: %w", err)
	}

	cfg.Mirror.PullEvery = every

	if err := cfg.normalizeMirror(); err != nil {
		return Config{}, err
	}

	if err := cfg.normalizeAuthSessions(); err != nil {
		return Config{}, err
	}

	nets, err := netutil.ParseCIDRs(cfg.HTTP.TrustedProxies)
	if err != nil {
		return Config{}, fmt.Errorf("http.trusted_proxies: %w", err)
	}
	cfg.TrustedProxyNets = nets

	return cfg, nil
}

func (c *Config) normalizeMirror() error {
	if c.Mirror.Workers <= 0 {
		c.Mirror.Workers = DefaultMirrorWorkers
	}

	if c.Mirror.DefaultMaxBytes < 0 {
		return fmt.Errorf("mirror.default_max_bytes: must be >= 0")
	}

	if c.Mirror.DefaultMaxBytes == 0 {
		c.Mirror.DefaultMaxBytes = DefaultMirrorMaxBytes
	}

	var err error
	if c.Mirror.CloneTimeoutDur, err = parseDurationDefault("mirror.clone_timeout", c.Mirror.CloneTimeout, DefaultMirrorCloneTimeout); err != nil {
		return err
	}

	if c.Mirror.FetchTimeoutDur, err = parseDurationDefault("mirror.fetch_timeout", c.Mirror.FetchTimeout, DefaultMirrorFetchTimeout); err != nil {
		return err
	}

	if c.Mirror.PushTimeoutDur, err = parseDurationDefault("mirror.push_timeout", c.Mirror.PushTimeout, DefaultMirrorPushTimeout); err != nil {
		return err
	}

	if c.Server.SoftDeletedRetentionDur, err = parseDurationDefault("server.soft_deleted_retention", c.Server.SoftDeletedRetention, DefaultSoftDeletedRetention); err != nil {
		return err
	}

	return nil
}

func (c *Config) normalizeAuthSessions() error {
	var err error
	if c.Auth.SessionAbsoluteDur, err = parseDurationDefault("auth.session_absolute", c.Auth.SessionAbsolute, DefaultSessionAbsolute); err != nil {
		return err
	}

	if c.Auth.SessionIdleDur, err = parseDurationDefault("auth.session_idle", c.Auth.SessionIdle, DefaultSessionIdle); err != nil {
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
	secrets := []struct{ name, val string }{
		{"secrets.session_secret", c.Secrets.SessionSecret},
		{"secrets.token_pepper", c.Secrets.TokenPepper},
		{"secrets.csrf_secret", c.Secrets.CSRFSecret},
		{"secrets.mirror_secrets_key", c.Secrets.MirrorSecretsKey},
	}
	for _, s := range secrets {
		if len(strings.TrimSpace(s.val)) < MinSecretLen {
			return fmt.Errorf("%s required in config (min %d characters)", s.name, MinSecretLen)
		}
	}

	for i := range secrets {
		for j := i + 1; j < len(secrets); j++ {
			if strings.TrimSpace(secrets[i].val) == strings.TrimSpace(secrets[j].val) {
				return fmt.Errorf("%s and %s must be different", secrets[i].name, secrets[j].name)
			}
		}
	}

	return nil
}

func (c Config) RequireAuthRead() bool {
	return c.Auth.RequireAuthForRead
}

func (c Config) ValidateListen() error {
	if !c.HTTP.CookieSecure && !c.listensLoopbackOnly() {
		return fmt.Errorf("http.cookie_secure must be true when http.addr %q is not loopback-only (serve over HTTPS, or bind 127.0.0.1 for local development)", c.HTTP.Addr)
	}

	return nil
}

func (c Config) WarnInsecure() []string {
	var out []string

	if c.Auth.AllowRegistration {
		out = append(out, "auth.allow_registration is true: anyone who can reach the server can create an account")
	}

	if c.Auth.AllowPublicContribute {
		out = append(out, "auth.allow_public_contribute is true: any logged-in user can open issues/PRs/comments on public repos")
	}

	if !c.HTTP.CookieSecure && !c.listensLoopbackOnly() {
		out = append(out, "http.cookie_secure is false while http.addr "+c.HTTP.Addr+" is not loopback-only: session cookies may be sent over plain HTTP; set http.cookie_secure: true behind HTTPS")
	}

	if len(c.HTTP.TrustedProxies) > 0 && !c.HTTP.CookieSecure {
		out = append(out, "http.trusted_proxies is set but http.cookie_secure is false: a reverse proxy normally terminates HTTPS, so session cookies would be issued without the Secure/__Host- protections; set http.cookie_secure: true")
	}

	if !c.RequireAuthRead() && !c.listensLoopbackOnly() {
		out = append(out, "auth.require_auth_for_read is false while http.addr "+c.HTTP.Addr+" is not loopback-only: anyone who can reach the server can read public repositories (REST and Git) anonymously; set auth.require_auth_for_read: true for a private instance")
	}

	if len(c.HTTP.TrustedProxies) == 0 && !c.listensLoopbackOnly() {
		out = append(out, "http.trusted_proxies is empty while http.addr "+c.HTTP.Addr+" is not loopback-only: X-Forwarded-For/Proto are ignored, so behind a reverse proxy every client shares the proxy IP (shared rate limits and lockouts, useless audit IPs); set http.trusted_proxies to the proxy address, or bind http.addr to 127.0.0.1")
	}

	for _, cidr := range c.HTTP.TrustedProxies {
		if ones, bits := proxyCIDRMaskSize(cidr); bits > 0 && ones < 8 {
			out = append(out, "http.trusted_proxies entry "+cidr+" is unusually broad (/"+strconv.Itoa(ones)+"): a client that can reach you may spoof X-Forwarded-For/Proto and bypass rate limits or Secure cookie checks; prefer the reverse-proxy address alone")
			break
		}
	}

	if c.Auth.GitPasswordAuth && !c.listensLoopbackOnly() {
		out = append(out, "auth.git_password_auth is true on a non-loopback listener: account passwords work on Git HTTP and ignore PAT repo_ids scopes; set auth.git_password_auth: false to require PATs only")
	}

	return out
}

func proxyCIDRMaskSize(cidr string) (ones, bits int) {
	_, n, err := net.ParseCIDR(strings.TrimSpace(cidr))
	if err != nil || n == nil {
		return 0, 0
	}

	ones, bits = n.Mask.Size()
	return ones, bits
}

func (c Config) listensLoopbackOnly() bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(c.HTTP.Addr))
	if err != nil {
		host = strings.TrimSpace(c.HTTP.Addr)
	}

	host = strings.Trim(host, "[]")
	if host == "" {
		return false
	}

	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (c Config) TokenPepperMaterial() string {
	return strings.TrimSpace(c.Secrets.TokenPepper)
}

func (c Config) CSRFSecretMaterial() string {
	return strings.TrimSpace(c.Secrets.CSRFSecret)
}

func (c Config) MirrorSecretsMaterial() string {
	return strings.TrimSpace(c.Secrets.MirrorSecretsKey)
}
