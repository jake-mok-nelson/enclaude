package cli

import (
	"fmt"
	"os"

	"github.com/jakenelson/enclaude/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "enclaude [flags] [-- claude-args...]",
	Short: "Run Claude Code in a secure container",
	Long: `Enclaude runs Claude Code in a secure, isolated Docker container with
filesystem isolation. Your working directory and credentials are mounted
automatically, while sensitive host files are protected.

Examples:
  enclaude                              # Run interactively in current directory
  enclaude -w ~/projects/myapp          # Override working directory
  enclaude -m ~/shared-lib              # Mount additional directory
  enclaude --mount-ro ~/docs            # Mount read-only
  enclaude --claude-auth=api-key        # Use API key auth only
  enclaude --no-external-credentials    # Disable GitHub/GCloud/SSH passthrough
  enclaude -- --help                    # Pass args to Claude Code`,
	RunE:         runContainer,
	SilenceUsage: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/enclaude/config.yaml)")

	// Run flags
	rootCmd.Flags().StringP("workdir", "w", "", "working directory to mount (default: current directory)")
	rootCmd.Flags().StringArrayP("mount", "m", nil, "additional directories to mount (read-write)")
	rootCmd.Flags().StringArray("mount-ro", nil, "additional directories to mount (read-only)")
	rootCmd.Flags().String("image", "", "Docker image to use (default: enclaude:latest)")

	// Claude authentication flags (override config)
	rootCmd.Flags().String("claude-auth", "", "Claude auth method: auto, session, api-key (overrides config)")
	rootCmd.Flags().String("claude-session-dir", "", "Session dir mode: none, readonly, readwrite (overrides config)")

	// External credentials flag
	rootCmd.Flags().Bool("no-external-credentials", false, "Disable external credential passthrough (GitHub, GCloud, SSH)")

	// Bind flags to viper for config integration
	viper.BindPFlag("image.name", rootCmd.Flags().Lookup("image"))
	viper.BindPFlag("claude.auth", rootCmd.Flags().Lookup("claude-auth"))
	viper.BindPFlag("claude.session_dir", rootCmd.Flags().Lookup("claude-session-dir"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Warning: could not find home directory:", err)
			return
		}

		// Search for config in standard locations
		viper.AddConfigPath(home + "/.config/enclaude")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("ENCLAUDE")
	viper.AutomaticEnv()

	// Read config file (ignore if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Warning: error reading config file:", err)
		}
	}

	// Load into config struct
	cfg = config.LoadConfig()
}
