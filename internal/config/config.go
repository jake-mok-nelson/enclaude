package config

import (
	"github.com/spf13/viper"
)

// Config represents the full configuration structure
type Config struct {
	Image       ImageConfig       `mapstructure:"image"`
	Mounts      MountsConfig      `mapstructure:"mounts"`
	Claude      ClaudeConfig      `mapstructure:"claude"`
	Credentials CredentialsConfig `mapstructure:"credentials"`
	Environment EnvironmentConfig `mapstructure:"environment"`
	Container   ContainerConfig   `mapstructure:"container"`
	Security    SecurityConfig    `mapstructure:"security"`
}

// ImageConfig configures the Docker image
type ImageConfig struct {
	Name         string `mapstructure:"name"`
	Dockerfile   string `mapstructure:"dockerfile"`
	BuildContext string `mapstructure:"build_context"`
}

// MountsConfig configures default mount behavior
type MountsConfig struct {
	Defaults  []MountEntry `mapstructure:"defaults"`
	ClaudeDir string       `mapstructure:"claude_dir"` // Deprecated: use claude.session_dir
}

// MountEntry represents a single mount configuration
type MountEntry struct {
	Path     string `mapstructure:"path"`
	ReadOnly bool   `mapstructure:"readonly"`
}

// ClaudeConfig configures Claude authentication and behavior
type ClaudeConfig struct {
	Auth        string   `mapstructure:"auth"`        // auto, session, api-key
	SessionDir  string   `mapstructure:"session_dir"` // none, readonly, readwrite
	DefaultArgs []string `mapstructure:"default_args"`
}

// CredentialsConfig configures external service credential passthrough
type CredentialsConfig struct {
	GitHub string    `mapstructure:"github"` // auto, enabled, disabled
	GCloud string    `mapstructure:"gcloud"` // auto, enabled, disabled
	SSH    SSHConfig `mapstructure:"ssh"`
}

// SSHConfig configures SSH credential passthrough
type SSHConfig struct {
	Enabled         bool     `mapstructure:"enabled"`
	Keys            []string `mapstructure:"keys"`
	KnownHosts      bool     `mapstructure:"known_hosts"`
	AgentForwarding bool     `mapstructure:"agent_forwarding"`
}

// EnvironmentConfig configures environment variables
type EnvironmentConfig struct {
	Passthrough []string          `mapstructure:"passthrough"`
	Custom      map[string]string `mapstructure:"custom"`
}

// ContainerConfig configures container runtime settings
type ContainerConfig struct {
	User        string `mapstructure:"user"`         // auto, or uid:gid
	MemoryLimit string `mapstructure:"memory_limit"` // e.g., "4g"
	Network     string `mapstructure:"network"`      // bridge, none, host
}

// SecurityConfig configures security settings
type SecurityConfig struct {
	DropCapabilities bool     `mapstructure:"drop_capabilities"`
	NoNewPrivileges  bool     `mapstructure:"no_new_privileges"`
	ReadOnlyRoot     bool     `mapstructure:"read_only_root"`
	CACerts          []string `mapstructure:"ca_certs"` // Additional CA certificate paths to mount
}

// LoadConfig loads configuration from viper with defaults
func LoadConfig() *Config {
	setDefaults()

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		// Return defaults on error
		return defaultConfig()
	}

	// Migrate deprecated mounts.claude_dir to claude.session_dir
	if cfg.Claude.SessionDir == "" && cfg.Mounts.ClaudeDir != "" {
		cfg.Claude.SessionDir = cfg.Mounts.ClaudeDir
	}

	return cfg
}

func setDefaults() {
	// Image defaults
	viper.SetDefault("image.name", "enclaude:latest")
	viper.SetDefault("image.dockerfile", "")
	viper.SetDefault("image.build_context", "")

	// Mount defaults
	viper.SetDefault("mounts.defaults", []MountEntry{})

	// Claude authentication defaults
	viper.SetDefault("claude.auth", "auto")
	viper.SetDefault("claude.session_dir", "readonly")
	viper.SetDefault("claude.default_args", []string{})

	// External credential defaults
	viper.SetDefault("credentials.github", "auto")
	viper.SetDefault("credentials.gcloud", "auto")
	viper.SetDefault("credentials.ssh.enabled", false)
	viper.SetDefault("credentials.ssh.keys", []string{})
	viper.SetDefault("credentials.ssh.known_hosts", true)
	viper.SetDefault("credentials.ssh.agent_forwarding", true)

	// Environment defaults
	viper.SetDefault("environment.passthrough", []string{"TERM", "COLORTERM", "EDITOR"})
	viper.SetDefault("environment.custom", map[string]string{})

	// Container defaults
	viper.SetDefault("container.user", "")
	viper.SetDefault("container.memory_limit", "4g")
	viper.SetDefault("container.network", "bridge")

	// Security defaults
	viper.SetDefault("security.drop_capabilities", true)
	viper.SetDefault("security.no_new_privileges", true)
	viper.SetDefault("security.read_only_root", true)
	viper.SetDefault("security.ca_certs", []string{})
}

func defaultConfig() *Config {
	return &Config{
		Image: ImageConfig{
			Name: "enclaude:latest",
		},
		Mounts: MountsConfig{
			Defaults: []MountEntry{},
		},
		Claude: ClaudeConfig{
			Auth:        "auto",
			SessionDir:  "readonly",
			DefaultArgs: []string{},
		},
		Credentials: CredentialsConfig{
			GitHub: "auto",
			GCloud: "auto",
			SSH: SSHConfig{
				Enabled:         false,
				Keys:            []string{},
				KnownHosts:      true,
				AgentForwarding: true,
			},
		},
		Environment: EnvironmentConfig{
			Passthrough: []string{"TERM", "COLORTERM", "EDITOR"},
			Custom:      map[string]string{},
		},
		Container: ContainerConfig{
			User:        "auto",
			MemoryLimit: "4g",
			Network:     "bridge",
		},
		Security: SecurityConfig{
			DropCapabilities: true,
			NoNewPrivileges:  true,
			ReadOnlyRoot:     true,
			CACerts:          []string{},
		},
	}
}
