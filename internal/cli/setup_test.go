package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectClaudeAuth(t *testing.T) {
	// Save original env var
	originalAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalAPIKey)
		}
	}()

	tests := []struct {
		name        string
		setupEnv    func(*testing.T) func()
		wantAPIKey  bool
		wantSession bool
	}{
		{
			name: "no auth methods",
			setupEnv: func(t *testing.T) func() {
				os.Unsetenv("ANTHROPIC_API_KEY")
				return func() {}
			},
			wantAPIKey:  false,
			wantSession: false,
		},
		{
			name: "api key only",
			setupEnv: func(t *testing.T) func() {
				os.Setenv("ANTHROPIC_API_KEY", "test-key")
				return func() {}
			},
			wantAPIKey:  true,
			wantSession: false,
		},
		{
			name: "session directory only",
			setupEnv: func(t *testing.T) func() {
				os.Unsetenv("ANTHROPIC_API_KEY")

				// Create a temporary session directory
				home, _ := os.UserHomeDir()
				claudePath := filepath.Join(home, ".claude")

				// Check if it already exists
				_, err := os.Stat(claudePath)
				alreadyExists := err == nil

				if !alreadyExists {
					os.MkdirAll(claudePath, 0755)
				}

				return func() {
					if !alreadyExists {
						os.RemoveAll(claudePath)
					}
				}
			},
			wantAPIKey:  false,
			wantSession: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := tt.setupEnv(t)
			defer cleanup()

			methods := detectClaudeAuth()

			if got := methods["api-key"]; got != tt.wantAPIKey {
				t.Errorf("detectClaudeAuth() api-key = %v, want %v", got, tt.wantAPIKey)
			}

			if got := methods["session"]; got != tt.wantSession {
				t.Errorf("detectClaudeAuth() session = %v, want %v", got, tt.wantSession)
			}
		})
	}
}

func TestGenerateConfig(t *testing.T) {
	config := generateConfig("auto", "auto", "disabled", false, "4g", "bridge")

	// Check that config contains expected values
	expectedStrings := []string{
		"auth: auto",
		"github: auto",
		"gcloud: disabled",
		"enabled: false",
		"memory_limit: 4g",
		"network: bridge",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(config, expected) {
			t.Errorf("generateConfig() missing expected string: %s", expected)
		}
	}
}
