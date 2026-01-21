package config

import (
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	// Test that CACerts is initialized to empty slice
	if cfg.Security.CACerts == nil {
		t.Error("defaultConfig().Security.CACerts should not be nil")
	}

	if len(cfg.Security.CACerts) != 0 {
		t.Errorf("defaultConfig().Security.CACerts should be empty, got %v", cfg.Security.CACerts)
	}

	// Test other security defaults
	if !cfg.Security.DropCapabilities {
		t.Error("defaultConfig().Security.DropCapabilities should be true")
	}

	if !cfg.Security.NoNewPrivileges {
		t.Error("defaultConfig().Security.NoNewPrivileges should be true")
	}

	if !cfg.Security.ReadOnlyRoot {
		t.Error("defaultConfig().Security.ReadOnlyRoot should be true")
	}
}

func TestSecurityConfigCACerts(t *testing.T) {
	// Test that SecurityConfig can hold CA certs
	cfg := SecurityConfig{
		DropCapabilities: true,
		NoNewPrivileges:  true,
		ReadOnlyRoot:     true,
		CACerts:          []string{"/path/to/cert1.crt", "/path/to/cert2.pem"},
	}

	if len(cfg.CACerts) != 2 {
		t.Errorf("expected 2 CA certs, got %d", len(cfg.CACerts))
	}

	if cfg.CACerts[0] != "/path/to/cert1.crt" {
		t.Errorf("expected '/path/to/cert1.crt', got '%s'", cfg.CACerts[0])
	}

	if cfg.CACerts[1] != "/path/to/cert2.pem" {
		t.Errorf("expected '/path/to/cert2.pem', got '%s'", cfg.CACerts[1])
	}
}
