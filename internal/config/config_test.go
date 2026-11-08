package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestLoadRejectsWorldOpenProxies(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("http:\n  trusted_proxies:\n    - \"0.0.0.0/0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsFlatKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("http_addr: \"127.0.0.1:8080\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("flat keys must be rejected")
	}
}

func TestLoadNested(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	raw := `
http:
  addr: "127.0.0.1:9090"
  cookie_secure: true
  trusted_proxies:
    - "10.0.0.1/32"
  receive_max_bytes: 1024
server:
  data_dir: "vardata"
  site_name: "Nest"
  soft_deleted_retention: 24h
secrets:
  session_secret: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  token_pepper: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  csrf_secret: "cccccccccccccccccccccccccccccccc"
  mirror_secrets_key: "dddddddddddddddddddddddddddddddd"
auth:
  allow_registration: true
  git_password_auth: false
  require_auth_for_read: true
mirror:
  pull_interval: 10m
  workers: 2
  verify:
    openpgp_keyring: "trust/a.asc"
    allowed_signers: "trust/signers"
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.HTTP.Addr != "127.0.0.1:9090" || !cfg.HTTP.CookieSecure || cfg.HTTP.ReceiveMaxBytes != 1024 {
		t.Fatalf("http: %+v", cfg.HTTP)
	}

	if cfg.Server.DataDir != "vardata" || cfg.Server.SiteName != "Nest" {
		t.Fatalf("server: %+v", cfg.Server)
	}

	if !cfg.Auth.AllowRegistration || cfg.Auth.GitPasswordAuth || !cfg.Auth.RequireAuthForRead {
		t.Fatalf("auth: %+v", cfg.Auth)
	}

	if cfg.Mirror.Workers != 2 || cfg.Mirror.Verify.OpenPGPKeyring != "trust/a.asc" {
		t.Fatalf("mirror: %+v", cfg.Mirror)
	}

	if err := cfg.ValidateSecrets(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadRejectsReadableConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("server:\n  site_name: x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("0600 must load: %v", err)
	}

	for _, mode := range []os.FileMode{0o640, 0o604, 0o644} {
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "chmod 600") {
			t.Fatalf("mode %04o must be rejected: %v", mode, err)
		}
	}

	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err != nil {
		t.Fatalf("0400 must load: %v", err)
	}
}

func TestValidateSecretsRequiresAllIndependent(t *testing.T) {
	t.Parallel()

	full := Config{
		Secrets: SecretsConfig{
			SessionSecret:    testSecret + "-session",
			TokenPepper:      testSecret + "-pepper",
			CSRFSecret:       testSecret + "-csrf",
			MirrorSecretsKey: testSecret + "-mirror",
		},
	}
	if err := full.ValidateSecrets(); err != nil {
		t.Fatal(err)
	}

	for name, mut := range map[string]func(*Config){
		"secrets.session_secret": func(c *Config) {
			c.Secrets.SessionSecret = "short"
		},
		"secrets.token_pepper": func(c *Config) {
			c.Secrets.TokenPepper = ""
		},
		"secrets.csrf_secret": func(c *Config) {
			c.Secrets.CSRFSecret = ""
		},
		"secrets.mirror_secrets_key": func(c *Config) {
			c.Secrets.MirrorSecretsKey = ""
		},
	} {
		c := full
		mut(&c)

		err := c.ValidateSecrets()
		if err == nil || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: err=%v", name, err)
		}
	}

	dup := full
	dup.Secrets.CSRFSecret = dup.Secrets.SessionSecret
	if err := dup.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "must be different") {
		t.Errorf("duplicate secrets must be rejected: %v", err)
	}
}

func TestSecretMaterialNoFallback(t *testing.T) {
	t.Parallel()

	c := Config{
		Secrets: SecretsConfig{SessionSecret: testSecret},
	}
	if c.MirrorSecretsMaterial() != "" || c.TokenPepperMaterial() != "" || c.CSRFSecretMaterial() != "" {
		t.Fatal("material must not fall back to session_secret")
	}
}

func TestWarnInsecure(t *testing.T) {
	t.Parallel()

	secure := Config{
		HTTP: HTTPConfig{
			Addr:           ":8080",
			CookieSecure:   true,
			TrustedProxies: []string{"10.0.0.1"},
		},
		Auth: AuthConfig{
			RequireAuthForRead: true,
		},
	}
	if w := secure.WarnInsecure(); len(w) != 0 {
		t.Fatalf("unexpected warnings: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr: "127.0.0.1:8080",
		},
	}).WarnInsecure(); len(w) != 0 {
		t.Fatalf("loopback: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr: "localhost:8080",
		},
	}).WarnInsecure(); len(w) != 0 {
		t.Fatalf("localhost: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
		Auth: AuthConfig{
			RequireAuthForRead: true,
		},
	}).WarnInsecure(); len(w) != 2 {
		t.Fatalf("public insecure cookie + no trusted_proxies: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr:         ":8080",
			CookieSecure: true,
		},
		Auth: AuthConfig{
			RequireAuthForRead: true,
		},
	}).WarnInsecure(); len(w) != 1 {
		t.Fatalf("secure cookie without trusted proxies: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr:           "127.0.0.1:8080",
			TrustedProxies: []string{"10.0.0.1"},
		},
		Auth: AuthConfig{RequireAuthForRead: true},
	}).WarnInsecure(); len(w) != 1 || !strings.Contains(w[0], "http.trusted_proxies is set") {
		t.Fatalf("trusted proxies without cookie_secure: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{
			Addr:           ":8080",
			CookieSecure:   true,
			TrustedProxies: []string{"10.0.0.1"},
		},
	}).WarnInsecure(); len(w) != 1 || !strings.Contains(w[0], "auth.require_auth_for_read") {
		t.Fatalf("anonymous git read on public listener: %v", w)
	}

	if w := (Config{
		HTTP: HTTPConfig{Addr: "[::1]:8080"},
		Auth: AuthConfig{AllowRegistration: true},
	}).WarnInsecure(); len(w) != 1 {
		t.Fatalf("registration: %v", w)
	}
}

func TestWarnInsecureTrustedProxiesIndependentOfCookieSecure(t *testing.T) {
	t.Parallel()

	for _, secure := range []bool{false, true} {
		w := (Config{
			HTTP: HTTPConfig{
				Addr:         ":8080",
				CookieSecure: secure,
			},
			Auth: AuthConfig{
				RequireAuthForRead: true,
			},
		}).WarnInsecure()

		found := false
		for _, line := range w {
			if strings.Contains(line, "http.trusted_proxies is empty") {
				found = true
			}
		}

		if !found {
			t.Fatalf("cookie_secure=%v: missing trusted_proxies warning: %v", secure, w)
		}
	}
}

func TestValidateListen(t *testing.T) {
	t.Parallel()

	if err := (Config{
		HTTP: HTTPConfig{
			Addr: ":8080",
		},
	}).ValidateListen(); err == nil {
		t.Fatal("non-loopback without cookie_secure must be refused")
	}

	for _, c := range []Config{
		{HTTP: HTTPConfig{
			Addr:         ":8080",
			CookieSecure: true,
		}},
		{HTTP: HTTPConfig{
			Addr: "127.0.0.1:8080",
		}},
		{HTTP: HTTPConfig{
			Addr: "[::1]:8080",
		}},
		{HTTP: HTTPConfig{
			Addr: "localhost:8080",
		}},
	} {
		if err := c.ValidateListen(); err != nil {
			t.Fatalf("%+v: %v", c, err)
		}
	}
}

func TestRequireAuthReadDefaults(t *testing.T) {
	t.Parallel()

	if (Config{}).RequireAuthRead() {
		t.Fatal("default must allow anonymous public reads")
	}

	if !(Config{
		Auth: AuthConfig{
			RequireAuthForRead: true,
		},
	}).RequireAuthRead() {
		t.Fatal("auth.require_auth_for_read must enable require-auth-read")
	}

	d := Default()
	if d.HTTP.Addr != "127.0.0.1:8080" || !d.Auth.GitPasswordAuth {
		t.Fatalf("unexpected defaults: %+v", d)
	}
}
