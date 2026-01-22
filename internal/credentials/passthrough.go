package credentials

import (
	"os"
	"path/filepath"

	"github.com/jakenelson/enclaude/internal/config"
	"github.com/jakenelson/enclaude/internal/container"
	"github.com/jakenelson/enclaude/internal/security"
)

// CollectClaudeAuth handles Claude Code authentication based on config.
// Returns mounts for ~/.claude session directory and environment variables for API key.
func CollectClaudeAuth(cfg *config.Config) ([]container.Mount, map[string]string) {
	var mounts []container.Mount
	env := make(map[string]string)

	home, err := os.UserHomeDir()
	if err != nil {
		return mounts, env
	}

	auth := cfg.Claude.Auth
	if auth == "" {
		auth = "auto"
	}

	// Handle API key
	if auth == "auto" || auth == "api-key" {
		if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
			env["ANTHROPIC_API_KEY"] = key
		}
	}

	// Handle session directory
	if auth == "auto" || auth == "session" {
		sessionDir := cfg.Claude.SessionDir
		if sessionDir == "" {
			sessionDir = "readonly"
		}
		if sessionDir != "none" {
			claudePath := filepath.Join(home, ".claude")
			if dirExists(claudePath) {
				// Mount to /tmp/.claude because container HOME is set to /tmp
				// This allows Claude to find the session directory while running as non-root
				mounts = append(mounts, container.Mount{
					Source:   claudePath,
					Target:   "/tmp/.claude",
					ReadOnly: sessionDir == "readonly",
				})
			}
		}
	}

	return mounts, env
}

// CollectExternalCredentials gathers external service credentials (GitHub, GCloud, SSH).
// This does not include Claude authentication - use CollectClaudeAuth for that.
func CollectExternalCredentials(cfg *config.Config) ([]container.Mount, map[string]string, error) {
	var mounts []container.Mount
	env := make(map[string]string)

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}

	// GitHub credentials
	if shouldEnable(cfg.Credentials.GitHub, "GH_TOKEN", "GITHUB_TOKEN") {
		// Try environment variable first
		if token := os.Getenv("GH_TOKEN"); token != "" {
			env["GH_TOKEN"] = token
		} else if token := os.Getenv("GITHUB_TOKEN"); token != "" {
			env["GH_TOKEN"] = token
		} else {
			// Try mounting gh config
			ghConfigPath := filepath.Join(home, ".config", "gh", "hosts.yml")
			if fileExists(ghConfigPath) {
				mounts = append(mounts, container.Mount{
					Source:   ghConfigPath,
					Target:   "/root/.config/gh/hosts.yml",
					ReadOnly: true,
				})
			}
		}
	}

	// Google Cloud ADC
	if shouldEnable(cfg.Credentials.GCloud, "GOOGLE_APPLICATION_CREDENTIALS") {
		adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
		if fileExists(adcPath) {
			mounts = append(mounts, container.Mount{
				Source:   adcPath,
				Target:   "/root/.config/gcloud/application_default_credentials.json",
				ReadOnly: true,
			})
			// Set the env var to point to the mounted location
			env["GOOGLE_APPLICATION_CREDENTIALS"] = "/root/.config/gcloud/application_default_credentials.json"
		}

		// Also check for explicit GOOGLE_APPLICATION_CREDENTIALS path
		if customPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); customPath != "" && fileExists(customPath) {
			mounts = append(mounts, container.Mount{
				Source:   customPath,
				Target:   "/root/.config/gcloud/application_default_credentials.json",
				ReadOnly: true,
			})
			env["GOOGLE_APPLICATION_CREDENTIALS"] = "/root/.config/gcloud/application_default_credentials.json"
		}
	}

	// SSH credentials (explicit opt-in)
	if cfg.Credentials.SSH.Enabled {
		sshMounts, sshEnv := collectSSHCredentials(cfg, home)
		mounts = append(mounts, sshMounts...)
		for k, v := range sshEnv {
			env[k] = v
		}
	}

	return mounts, env, nil
}

func collectSSHCredentials(cfg *config.Config, home string) ([]container.Mount, map[string]string) {
	var mounts []container.Mount
	env := make(map[string]string)

	// Mount specific SSH keys (read-only)
	for _, keyPath := range cfg.Credentials.SSH.Keys {
		expanded, err := security.ExpandPath(keyPath)
		if err != nil {
			// Skip keys with expansion errors
			continue
		}
		if fileExists(expanded) {
			// Determine target path
			keyName := filepath.Base(expanded)
			mounts = append(mounts, container.Mount{
				Source:   expanded,
				Target:   filepath.Join("/root/.ssh", keyName),
				ReadOnly: true,
			})
		}
	}

	// Mount known_hosts if configured
	if cfg.Credentials.SSH.KnownHosts {
		knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
		if fileExists(knownHostsPath) {
			mounts = append(mounts, container.Mount{
				Source:   knownHostsPath,
				Target:   "/root/.ssh/known_hosts",
				ReadOnly: true,
			})
		}
	}

	// SSH agent forwarding
	if cfg.Credentials.SSH.AgentForwarding {
		if authSock := os.Getenv("SSH_AUTH_SOCK"); authSock != "" {
			// On macOS with Docker Desktop, we need to use a special socket path
			// The socket forwarding is handled automatically by Docker Desktop
			mounts = append(mounts, container.Mount{
				Source:   authSock,
				Target:   "/tmp/ssh-agent.sock",
				ReadOnly: false,
			})
			env["SSH_AUTH_SOCK"] = "/tmp/ssh-agent.sock"
		}
	}

	return mounts, env
}

// shouldEnable determines if a credential should be enabled based on config and presence
func shouldEnable(setting string, envVars ...string) bool {
	switch setting {
	case "enabled":
		return true
	case "disabled":
		return false
	case "auto":
		// Auto-detect: enabled if any of the env vars are set or related files exist
		for _, v := range envVars {
			if os.Getenv(v) != "" {
				return true
			}
		}
		return true // Default to trying to pass through
	default:
		return true // Default to auto behavior
	}
}

// pathExists checks if a path exists and matches the expected type.
// If expectDir is true, checks for directory; if false, checks for file.
func pathExists(path string, expectDir bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir() == expectDir
}

func fileExists(path string) bool {
	return pathExists(path, false)
}

func dirExists(path string) bool {
	return pathExists(path, true)
}
