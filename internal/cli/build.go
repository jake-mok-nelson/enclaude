package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jakenelson/enclaude/internal/container"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(buildCmd)

	buildCmd.Flags().StringP("file", "f", "", "path to Dockerfile (default: built-in)")
	buildCmd.Flags().StringP("tag", "t", "enclaude:latest", "image tag")
	buildCmd.Flags().String("context", "", "build context directory")
	buildCmd.Flags().Bool("no-cache", false, "do not use cache when building")
	buildCmd.Flags().String("platform", "", "target platform (e.g., linux/amd64,linux/arm64)")
}

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build the enclaude Docker image",
	Long: `Build the enclaude Docker image from the built-in Dockerfile or a custom one.

Examples:
  enclaude build                        # Build with default settings
  enclaude build -t my-enclaude:v1      # Custom tag
  enclaude build -f ./Dockerfile.custom # Use custom Dockerfile`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		dockerfile, _ := cmd.Flags().GetString("file")
		tag, _ := cmd.Flags().GetString("tag")
		contextDir, _ := cmd.Flags().GetString("context")
		noCache, _ := cmd.Flags().GetBool("no-cache")
		platform, _ := cmd.Flags().GetString("platform")

		// Use config values if flags not provided
		if dockerfile == "" && cfg.Image.Dockerfile != "" {
			dockerfile = cfg.Image.Dockerfile
		}
		if contextDir == "" && cfg.Image.BuildContext != "" {
			contextDir = cfg.Image.BuildContext
		}

		// If no dockerfile specified, look for built-in one
		if dockerfile == "" {
			// Check common locations
			locations := []string{
				"docker/Dockerfile",
				"Dockerfile",
			}

			// Also check relative to executable
			if execPath, err := os.Executable(); err == nil {
				execDir := filepath.Dir(execPath)
				locations = append([]string{
					filepath.Join(execDir, "docker", "Dockerfile"),
					filepath.Join(execDir, "..", "docker", "Dockerfile"),
				}, locations...)
			}

			for _, loc := range locations {
				if _, err := os.Stat(loc); err == nil {
					dockerfile = loc
					break
				}
			}

			if dockerfile == "" {
				return fmt.Errorf("no Dockerfile found; use -f to specify one or run from the enclaude source directory")
			}
		}

		// Default context to Dockerfile directory
		if contextDir == "" {
			contextDir = filepath.Dir(dockerfile)
		}

		runner, err := container.NewRunner()
		if err != nil {
			return fmt.Errorf("failed to create container runner: %w", err)
		}
		defer runner.Close()

		opts := container.BuildOptions{
			Dockerfile: dockerfile,
			ContextDir: contextDir,
			Tag:        tag,
			NoCache:    noCache,
			Platform:   platform,
		}

		fmt.Printf("Building image %s from %s...\n", tag, dockerfile)
		if err := runner.Build(ctx, opts); err != nil {
			return fmt.Errorf("build failed: %w", err)
		}

		fmt.Printf("Successfully built %s\n", tag)
		return nil
	},
}
