package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for enclaude configuration",
	Long: `Interactive setup wizard that guides you through configuring enclaude
and detects available Claude authentication methods on your system.

This command will:
- Detect available Claude authentication methods (API key, session directory)
- Guide you through selecting authentication preferences
- Configure external credential passthrough (GitHub, GCloud, SSH)
- Create or update your configuration file
- Verify the Docker image is available

Run this command when first installing enclaude or to reconfigure settings.`,
	RunE: runSetup,
}

func runSetup(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("🔧 Enclaude Setup Wizard")
	fmt.Println("========================")

	// Step 1: Detect Claude authentication
	fmt.Println("Step 1: Detecting Claude Authentication Methods")
	fmt.Println("-----------------------------------------------")
	authMethods := detectClaudeAuth()
	displayAuthMethods(authMethods)

	// Step 2: Select authentication method
	fmt.Println("\nStep 2: Configure Claude Authentication")
	fmt.Println("----------------------------------------")
	selectedAuth := selectAuthMethod(reader, authMethods)

	// Step 3: Configure external credentials
	fmt.Println("\nStep 3: Configure External Credentials")
	fmt.Println("---------------------------------------")
	githubCred := configureCredential(reader, "GitHub", "auto")
	gcloudCred := configureCredential(reader, "Google Cloud", "auto")
	sshEnabled := configureSSH(reader)

	// Step 4: Container preferences
	fmt.Println("\nStep 4: Container Preferences")
	fmt.Println("-----------------------------")
	memoryLimit := configureMemory(reader)
	network := configureNetwork(reader)

	// Step 5: Create config file
	fmt.Println("\nStep 5: Creating Configuration")
	fmt.Println("------------------------------")
	configPath := getConfigPath()

	// Check if config exists
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
		fmt.Printf("⚠️  Configuration file already exists at: %s\n", configPath)
		if !confirm(reader, "Do you want to overwrite it?") {
			fmt.Println("\n❌ Setup cancelled. No changes were made.")
			return nil
		}
	}

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate config content
	configContent := generateConfig(selectedAuth, githubCred, gcloudCred, sshEnabled, memoryLimit, network)

	// Write config file
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if configExists {
		fmt.Printf("\n✅ Configuration updated at: %s\n", configPath)
	} else {
		fmt.Printf("\n✅ Configuration created at: %s\n", configPath)
	}

	// Step 6: Verify Docker image
	fmt.Println("\nStep 6: Docker Image")
	fmt.Println("--------------------")
	fmt.Println("📦 To use enclaude, you need the Docker image.")
	fmt.Println("   Run: enclaude build")
	fmt.Println("   Or use a custom image with: enclaude --image <image-name>")

	fmt.Println("\n✨ Setup complete! You can now run 'enclaude' to start.")
	fmt.Println("   Use 'enclaude config list' to view your configuration.")

	return nil
}

// detectClaudeAuth detects available Claude authentication methods
func detectClaudeAuth() map[string]bool {
	methods := make(map[string]bool)

	// Check for API key
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		methods["api-key"] = true
	}

	// Check for session directory
	home, err := os.UserHomeDir()
	if err == nil {
		claudePath := filepath.Join(home, ".claude")
		if info, err := os.Stat(claudePath); err == nil && info.IsDir() {
			methods["session"] = true
		}
	}

	return methods
}

// displayAuthMethods shows detected authentication methods
func displayAuthMethods(methods map[string]bool) {
	if len(methods) == 0 {
		fmt.Println("⚠️  No Claude authentication methods detected.")
		fmt.Println("   You can still configure enclaude and set up authentication later.")
		return
	}

	fmt.Println("✅ Detected authentication methods:")
	if methods["api-key"] {
		fmt.Println("   • API Key (ANTHROPIC_API_KEY environment variable)")
	}
	if methods["session"] {
		fmt.Println("   • Session Directory (~/.claude)")
	}
}

// selectAuthMethod prompts user to select authentication method
func selectAuthMethod(reader *bufio.Reader, methods map[string]bool) string {
	fmt.Println("\nSelect Claude authentication mode:")
	fmt.Println("  1) auto     - Use all available methods (recommended)")
	fmt.Println("  2) api-key  - Use API key only")
	fmt.Println("  3) session  - Use session directory only")

	// Determine default based on what's available
	defaultChoice := "auto"

	for {
		fmt.Printf("\nChoice [1-3] (default: auto): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\nError reading input: %v\n", err)
			return defaultChoice
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultChoice
		}

		switch input {
		case "1":
			return "auto"
		case "2":
			if !methods["api-key"] {
				fmt.Println("⚠️  API key not detected. You can still select this option.")
			}
			return "api-key"
		case "3":
			if !methods["session"] {
				fmt.Println("⚠️  Session directory not detected. You can still select this option.")
			}
			return "session"
		default:
			fmt.Println("❌ Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

// configureCredential prompts for credential configuration
func configureCredential(reader *bufio.Reader, name, defaultValue string) string {
	fmt.Printf("\nConfigure %s credentials:\n", name)
	fmt.Println("  1) auto     - Auto-detect and use if available")
	fmt.Println("  2) enabled  - Always enable (will fail if not available)")
	fmt.Println("  3) disabled - Never use")

	for {
		fmt.Printf("\nChoice [1-3] (default: auto): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\nError reading input: %v\n", err)
			return defaultValue
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultValue
		}

		switch input {
		case "1":
			return "auto"
		case "2":
			return "enabled"
		case "3":
			return "disabled"
		default:
			fmt.Println("❌ Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

// configureSSH prompts for SSH configuration
func configureSSH(reader *bufio.Reader) bool {
	fmt.Println("\nConfigure SSH credentials:")
	fmt.Println("  SSH credentials are disabled by default for security.")
	fmt.Println("  Enable if you need to use SSH keys or agent forwarding.")
	return confirm(reader, "Enable SSH credentials?")
}

// configureMemory prompts for memory limit
func configureMemory(reader *bufio.Reader) string {
	fmt.Println("\nContainer memory limit:")
	fmt.Println("  Set the maximum memory for the container (e.g., 2g, 4g, 8g)")

	for {
		fmt.Printf("Memory limit (default: 4g): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\nError reading input: %v\n", err)
			return "4g"
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return "4g"
		}

		// Basic validation
		if len(input) >= 2 && (strings.HasSuffix(input, "g") || strings.HasSuffix(input, "m")) {
			return input
		}

		fmt.Println("❌ Invalid format. Use format like '4g' or '512m'.")
	}
}

// configureNetwork prompts for network mode
func configureNetwork(reader *bufio.Reader) string {
	fmt.Println("\nContainer network mode:")
	fmt.Println("  1) bridge - Standard Docker bridge network (recommended)")
	fmt.Println("  2) host   - Use host network (less isolated)")
	fmt.Println("  3) none   - No network access")

	for {
		fmt.Printf("\nChoice [1-3] (default: bridge): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\nError reading input: %v\n", err)
			return "bridge"
		}
		input = strings.TrimSpace(input)

		if input == "" {
			return "bridge"
		}

		switch input {
		case "1":
			return "bridge"
		case "2":
			return "host"
		case "3":
			return "none"
		default:
			fmt.Println("❌ Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}

// confirm prompts for yes/no confirmation
func confirm(reader *bufio.Reader, prompt string) bool {
	for {
		fmt.Printf("%s [y/N]: ", prompt)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("\nError reading input: %v\n", err)
			return false
		}
		input = strings.ToLower(strings.TrimSpace(input))

		if input == "" || input == "n" || input == "no" {
			return false
		}
		if input == "y" || input == "yes" {
			return true
		}

		fmt.Println("❌ Please enter 'y' or 'n'.")
	}
}

// generateConfig creates the configuration file content
func generateConfig(auth, github, gcloud string, sshEnabled bool, memory, network string) string {
	sshEnabledStr := "false"
	if sshEnabled {
		sshEnabledStr = "true"
	}

	return fmt.Sprintf(`# Enclaude configuration
# Generated by 'enclaude setup'
# See https://github.com/jakenelson/enclaude for documentation

# Image settings
image:
  name: enclaude:latest

# Default mounts (in addition to working directory)
mounts:
  defaults: []

# Claude Code authentication
claude:
  auth: %s              # auto | session | api-key
  session_dir: readonly   # none | readonly | readwrite
  default_args: []

# External service credentials
credentials:
  github: %s       # auto | enabled | disabled
  gcloud: %s       # auto | enabled | disabled
  ssh:
    enabled: %s   # Explicit opt-in for SSH
    keys: []         # Specific keys to mount (read-only)
    known_hosts: true       # Include ~/.ssh/known_hosts
    agent_forwarding: true  # Forward SSH_AUTH_SOCK

# Environment variables to pass through
environment:
  passthrough:
    - TERM
    - COLORTERM
    - EDITOR
  custom: {}

# Container settings
container:
  user: auto          # auto | uid:gid
  memory_limit: %s
  network: %s     # bridge | none | host

# Security settings
security:
  drop_capabilities: true
  no_new_privileges: true
  read_only_root: true
  ca_certs: []        # Additional CA certificates to mount (e.g., corporate CA)
`, auth, github, gcloud, sshEnabledStr, memory, network)
}
