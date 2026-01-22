package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jakenelson/enclaude/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configInitCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage enclaude configuration",
	Long: `Manage enclaude configuration settings.

Commands:
  list    List all configuration settings
  get     Get a configuration value
  set     Set a configuration value
  path    Show configuration file path
  init    Create default configuration file

Examples:
  enclaude config list
  enclaude config get claude.auth
  enclaude config set claude.auth api-key
  enclaude config set credentials.github disabled`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		settings := viper.AllSettings()
		printSettingsFlat("", settings)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		if !viper.IsSet(key) {
			return fmt.Errorf("key not found: %s", key)
		}
		value := viper.Get(key)
		// Handle nested maps by printing them in a readable format
		if m, ok := value.(map[string]interface{}); ok {
			printSettingsFlat(key, m)
		} else {
			fmt.Println(value)
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key, value := args[0], args[1]

		// Validate known keys
		if err := validateConfigKey(key, value); err != nil {
			return err
		}

		// Get config file path
		configPath := getConfigPath()

		// Ensure config directory exists
		configDir := filepath.Dir(configPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Parse value (handle booleans)
		var parsedValue interface{} = value
		if value == "true" {
			parsedValue = true
		} else if value == "false" {
			parsedValue = false
		}

		// Update the value
		viper.Set(key, parsedValue)

		// Write config to file
		if err := viper.WriteConfigAs(configPath); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Show configuration file path",
	Run: func(cmd *cobra.Command, args []string) {
		if cfgFile := viper.ConfigFileUsed(); cfgFile != "" {
			fmt.Println(cfgFile)
		} else {
			fmt.Println(getConfigPath())
		}
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := getConfigPath()
		configDir := filepath.Dir(configPath)

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("config file already exists at %s", configPath)
		}

		defaultConfig := `# Enclaude configuration
# See https://github.com/jakenelson/enclaude for documentation

# Image settings
image:
  name: enclaude:latest
  # dockerfile: ""       # Path to custom Dockerfile (optional)
  # build_context: ""    # Custom build context (optional)

# Default mounts (in addition to working directory)
mounts:
  defaults: []
    # - path: ~/projects/shared-utils
    #   readonly: true

# Claude Code authentication
claude:
  auth: auto              # auto | session | api-key
  session_dir: readwrite  # none | readonly | readwrite
  default_args: []
    # Example: ["--model", "claude-sonnet-4-20250514"]

# External service credentials
credentials:
  github: auto       # auto | enabled | disabled
  gcloud: auto       # auto | enabled | disabled
  ssh:
    enabled: false   # Explicit opt-in for SSH
    keys: []         # Specific keys to mount (read-only)
      # - ~/.ssh/id_ed25519
      # - ~/.ssh/id_ed25519.pub
    known_hosts: true       # Include ~/.ssh/known_hosts
    agent_forwarding: true  # Forward SSH_AUTH_SOCK

# Environment variables to pass through
environment:
  passthrough:
    - TERM
    - COLORTERM
    - EDITOR
  custom: {}
    # DEBUG: "false"

# Container settings
container:
  user: auto          # auto | uid:gid
  memory_limit: 4g
  network: bridge     # bridge | none | host

# Security settings
security:
  drop_capabilities: true
  no_new_privileges: true
  read_only_root: true
`

		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}

		fmt.Printf("Created config file at %s\n", configPath)
		return nil
	},
}

// printSettingsFlat prints settings in dot notation
func printSettingsFlat(prefix string, settings map[string]interface{}) {
	// Collect keys and sort them for consistent output
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := settings[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		if nested, ok := value.(map[string]interface{}); ok {
			printSettingsFlat(fullKey, nested)
		} else {
			fmt.Printf("%s: %v\n", fullKey, value)
		}
	}
}

// getConfigPath returns the default config file path
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "enclaude", "config.yaml")
}

// validateConfigKey validates key/value pairs for known configuration keys
func validateConfigKey(key, value string) error {
	validations := map[string][]string{
		"claude.auth":        {config.AuthAuto, config.AuthSession, config.AuthAPIKey},
		"claude.session_dir": {config.SessionNone, config.SessionReadOnly, config.SessionReadWrite},
		"credentials.github": {config.CredentialAuto, config.CredentialEnabled, config.CredentialDisabled},
		"credentials.gcloud": {config.CredentialAuto, config.CredentialEnabled, config.CredentialDisabled},
		"container.network":  {config.NetworkBridge, config.NetworkNone, config.NetworkHost},
	}

	if allowed, exists := validations[key]; exists {
		for _, v := range allowed {
			if value == v {
				return nil
			}
		}
		return fmt.Errorf("invalid value for %s: %s (allowed: %s)", key, value, strings.Join(allowed, ", "))
	}
	return nil // Unknown keys pass through
}
