package credentials

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jakenelson/enclaude/internal/config"
)

func TestCollectClaudeAuth_SessionDirectory(t *testing.T) {
	// Create a temporary .claude directory in the user's home
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	claudePath := filepath.Join(home, ".claude")
	_, err = os.Stat(claudePath)
	alreadyExists := !os.IsNotExist(err)

	if !alreadyExists {
		if err := os.MkdirAll(claudePath, 0755); err != nil {
			t.Fatalf("failed to create .claude directory: %v", err)
		}
		defer os.RemoveAll(claudePath)
	}

	tests := []struct {
		name           string
		sessionDir     string
		wantTarget     string
		wantReadOnly   bool
		wantMountCount int
	}{
		{
			name:           "default (empty) should be readonly",
			sessionDir:     "",
			wantTarget:     "/tmp/.claude",
			wantReadOnly:   true,
			wantMountCount: 1,
		},
		{
			name:           "explicit readonly",
			sessionDir:     "readonly",
			wantTarget:     "/tmp/.claude",
			wantReadOnly:   true,
			wantMountCount: 1,
		},
		{
			name:           "explicit readwrite",
			sessionDir:     "readwrite",
			wantTarget:     "/tmp/.claude",
			wantReadOnly:   false,
			wantMountCount: 1,
		},
		{
			name:           "none should not mount",
			sessionDir:     "none",
			wantTarget:     "",
			wantReadOnly:   false,
			wantMountCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Claude: config.ClaudeConfig{
					Auth:       "auto",
					SessionDir: tt.sessionDir,
				},
			}

			mounts, env := CollectClaudeAuth(cfg)

			// Verify no unexpected API key was set (since we didn't set ANTHROPIC_API_KEY)
			if _, hasAPIKey := env["ANTHROPIC_API_KEY"]; hasAPIKey && os.Getenv("ANTHROPIC_API_KEY") == "" {
				t.Errorf("CollectClaudeAuth() unexpectedly has API key in env")
			}

			if len(mounts) != tt.wantMountCount {
				t.Errorf("CollectClaudeAuth() mount count = %d, want %d", len(mounts), tt.wantMountCount)
				return
			}

			if tt.wantMountCount > 0 {
				mount := mounts[0]

				if mount.Target != tt.wantTarget {
					t.Errorf("CollectClaudeAuth() target = %s, want %s", mount.Target, tt.wantTarget)
				}

				if mount.ReadOnly != tt.wantReadOnly {
					t.Errorf("CollectClaudeAuth() readonly = %v, want %v", mount.ReadOnly, tt.wantReadOnly)
				}

				// Verify source is the host's .claude directory
				expectedSource := filepath.Join(home, ".claude")
				if mount.Source != expectedSource {
					t.Errorf("CollectClaudeAuth() source = %s, want %s", mount.Source, expectedSource)
				}
			}
		})
	}
}

func TestCollectClaudeAuth_APIKey(t *testing.T) {
	// Save and restore original API key
	originalAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		if originalAPIKey == "" {
			os.Unsetenv("ANTHROPIC_API_KEY")
		} else {
			os.Setenv("ANTHROPIC_API_KEY", originalAPIKey)
		}
	}()

	tests := []struct {
		name       string
		auth       string
		apiKey     string
		wantAPIKey bool
	}{
		{
			name:       "auto with API key",
			auth:       "auto",
			apiKey:     "test-key",
			wantAPIKey: true,
		},
		{
			name:       "api-key mode with API key",
			auth:       "api-key",
			apiKey:     "test-key",
			wantAPIKey: true,
		},
		{
			name:       "session mode ignores API key",
			auth:       "session",
			apiKey:     "test-key",
			wantAPIKey: false,
		},
		{
			name:       "auto without API key",
			auth:       "auto",
			apiKey:     "",
			wantAPIKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.apiKey != "" {
				os.Setenv("ANTHROPIC_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("ANTHROPIC_API_KEY")
			}

			cfg := &config.Config{
				Claude: config.ClaudeConfig{
					Auth:       tt.auth,
					SessionDir: "none", // Disable session dir for this test
				},
			}

			_, env := CollectClaudeAuth(cfg)

			_, hasAPIKey := env["ANTHROPIC_API_KEY"]
			if hasAPIKey != tt.wantAPIKey {
				t.Errorf("CollectClaudeAuth() has API key = %v, want %v", hasAPIKey, tt.wantAPIKey)
			}
		})
	}
}
