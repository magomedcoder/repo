package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestParseCIDRsRejectsWorldOpen(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"0.0.0.0/0", "::/0"} {
		if _, err := parseCIDRs([]string{raw}); err == nil {
			t.Fatalf("%s must be rejected", raw)
		}
	}

	nets, err := parseCIDRs([]string{"127.0.0.1", "10.0.0.0/8", "::1/128"})
	if err != nil || len(nets) != 3 {
		t.Fatalf("nets=%v err=%v", nets, err)
	}
}

func TestLoadRejectsWorldOpenProxies(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("trusted_proxies:\n  - \"0.0.0.0/0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateSecretsRequiresAllIndependent(t *testing.T) {
	t.Parallel()

	full := Config{
		SessionSecret:    testSecret + "-session",
		TokenPepper:      testSecret + "-pepper",
		CSRFSecret:       testSecret + "-csrf",
		MirrorSecretsKey: testSecret + "-mirror",
	}
	if err := full.ValidateSecrets(); err != nil {
		t.Fatal(err)
	}

	for name, mut := range map[string]func(*Config){
		"session_secret": func(c *Config) {
			c.SessionSecret = "short"
		},
		"token_pepper": func(c *Config) {
			c.TokenPepper = ""
		},
		"csrf_secret": func(c *Config) {
			c.CSRFSecret = ""
		},
		"mirror_secrets_key": func(c *Config) {
			c.MirrorSecretsKey = ""
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
	dup.CSRFSecret = dup.SessionSecret
	if err := dup.ValidateSecrets(); err == nil || !strings.Contains(err.Error(), "must be different") {
		t.Errorf("duplicate secrets must be rejected: %v", err)
	}
}

func TestSecretMaterialNoFallback(t *testing.T) {
	t.Parallel()

	c := Config{
		SessionSecret: testSecret,
	}
	if c.MirrorSecretsMaterial() != "" || c.TokenPepperMaterial() != "" || c.CSRFSecretMaterial() != "" {
		t.Fatal("material must not fall back to session_secret")
	}
}

func TestWarnInsecure(t *testing.T) {
	t.Parallel()

	secure := Config{
		HTTPAddr:       ":8080",
		CookieSecure:   true,
		TrustedProxies: []string{"10.0.0.1"},
	}
	if w := secure.WarnInsecure(); len(w) != 0 {
		t.Fatalf("unexpected warnings: %v", w)
	}

	if w := (Config{
		HTTPAddr: "127.0.0.1:8080",
	}).WarnInsecure(); len(w) != 0 {
		t.Fatalf("loopback: %v", w)
	}

	if w := (Config{
		HTTPAddr: "localhost:8080",
	}).WarnInsecure(); len(w) != 0 {
		t.Fatalf("localhost: %v", w)
	}

	if w := (Config{
		HTTPAddr: ":8080",
	}).WarnInsecure(); len(w) != 1 {
		t.Fatalf("public insecure cookie: %v", w)
	}

	if w := (Config{
		HTTPAddr:     ":8080",
		CookieSecure: true,
	}).WarnInsecure(); len(w) != 1 {
		t.Fatalf("secure cookie without trusted proxies: %v", w)
	}

	if w := (Config{
		HTTPAddr:          "[::1]:8080",
		AllowRegistration: true,
	}).WarnInsecure(); len(w) != 1 {
		t.Fatalf("registration: %v", w)
	}
}
