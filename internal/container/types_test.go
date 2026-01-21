package container

import (
	"testing"
)

func TestSecurityOptionsCACerts(t *testing.T) {
	// Test that SecurityOptions can hold CA certs
	opts := SecurityOptions{
		DropCapabilities: true,
		NoNewPrivileges:  true,
		ReadOnlyRoot:     true,
		CACerts:          []string{"/path/to/corporate-ca.crt"},
	}

	if len(opts.CACerts) != 1 {
		t.Errorf("expected 1 CA cert, got %d", len(opts.CACerts))
	}

	if opts.CACerts[0] != "/path/to/corporate-ca.crt" {
		t.Errorf("expected '/path/to/corporate-ca.crt', got '%s'", opts.CACerts[0])
	}
}

func TestSecurityOptionsDefaults(t *testing.T) {
	// Test empty SecurityOptions
	opts := SecurityOptions{}

	if opts.CACerts != nil && len(opts.CACerts) > 0 {
		t.Errorf("expected empty CACerts, got %v", opts.CACerts)
	}
}

func TestRunOptionsWithCACerts(t *testing.T) {
	// Test RunOptions with SecurityOptions containing CA certs
	opts := RunOptions{
		Image:   "enclaude:latest",
		WorkDir: "/workspace",
		Security: SecurityOptions{
			DropCapabilities: true,
			NoNewPrivileges:  true,
			ReadOnlyRoot:     true,
			CACerts:          []string{"/etc/ssl/custom/ca.crt", "/etc/ssl/custom/ca2.pem"},
		},
	}

	if len(opts.Security.CACerts) != 2 {
		t.Errorf("expected 2 CA certs, got %d", len(opts.Security.CACerts))
	}
}
