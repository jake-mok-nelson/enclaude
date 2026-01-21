package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/jakenelson/enclaude/internal/container"
	"github.com/jakenelson/enclaude/internal/credentials"
	"github.com/jakenelson/enclaude/internal/security"
	"github.com/spf13/cobra"
)

func runContainer(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Get working directory
	workDir, _ := cmd.Flags().GetString("workdir")
	if workDir == "" {
		var err error
		workDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Expand and validate working directory
	workDir, err := security.ExpandPath(workDir)
	if err != nil {
		return fmt.Errorf("invalid working directory: %w", err)
	}

	// Build mount configuration
	mounts := []container.Mount{
		{Source: workDir, Target: "/workspace", ReadOnly: false},
	}

	// Add additional mounts from flags
	extraMounts, _ := cmd.Flags().GetStringArray("mount")
	for _, m := range extraMounts {
		expanded, err := security.ExpandPath(m)
		if err != nil {
			return fmt.Errorf("invalid mount path %q: %w", m, err)
		}
		if err := security.ValidateMountPath(expanded); err != nil {
			return fmt.Errorf("mount path denied %q: %w", m, err)
		}
		mounts = append(mounts, container.Mount{Source: expanded, Target: expanded, ReadOnly: false})
	}

	// Add read-only mounts
	roMounts, _ := cmd.Flags().GetStringArray("mount-ro")
	for _, m := range roMounts {
		expanded, err := security.ExpandPath(m)
		if err != nil {
			return fmt.Errorf("invalid mount path %q: %w", m, err)
		}
		if err := security.ValidateMountPath(expanded); err != nil {
			return fmt.Errorf("mount path denied %q: %w", m, err)
		}
		mounts = append(mounts, container.Mount{Source: expanded, Target: expanded, ReadOnly: true})
	}

	// Add default mounts from config
	for _, dm := range cfg.Mounts.Defaults {
		expanded, err := security.ExpandPath(dm.Path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping invalid default mount %q: %v\n", dm.Path, err)
			continue
		}
		if err := security.ValidateMountPath(expanded); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping denied default mount %q: %v\n", dm.Path, err)
			continue
		}
		mounts = append(mounts, container.Mount{Source: expanded, Target: expanded, ReadOnly: dm.ReadOnly})
	}

	// Build environment variables
	env := make(map[string]string)

	// Passthrough environment variables from config
	for _, key := range cfg.Environment.Passthrough {
		if val, ok := os.LookupEnv(key); ok {
			env[key] = val
		}
	}

	// Custom environment variables from config
	for key, val := range cfg.Environment.Custom {
		env[key] = val
	}

	// Handle Claude authentication (always needed for Claude to work)
	claudeMounts, claudeEnv := credentials.CollectClaudeAuth(cfg)
	mounts = append(mounts, claudeMounts...)
	for k, v := range claudeEnv {
		env[k] = v
	}

	// Handle external credentials (unless disabled by flag)
	noExtCreds, _ := cmd.Flags().GetBool("no-external-credentials")
	if !noExtCreds {
		extMounts, extEnv, err := credentials.CollectExternalCredentials(cfg)
		if err != nil {
			return fmt.Errorf("failed to collect credentials: %w", err)
		}
		mounts = append(mounts, extMounts...)
		for k, v := range extEnv {
			env[k] = v
		}
	}

	// Get image name
	imageName, _ := cmd.Flags().GetString("image")
	if imageName == "" {
		imageName = cfg.Image.Name
	}

	// Expand and validate CA certificate paths
	var caCerts []string
	for _, certPath := range cfg.Security.CACerts {
		expanded, err := security.ExpandPath(certPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping invalid CA cert path %q: %v\n", certPath, err)
			continue
		}
		if err := security.ValidateMountPath(expanded); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: skipping denied CA cert path %q: %v\n", expanded, err)
			continue
		}
		if _, err := os.Stat(expanded); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: CA cert file not found %q\n", expanded)
			continue
		}
		caCerts = append(caCerts, expanded)
	}

	// Build run options
	opts := container.RunOptions{
		Image:       imageName,
		Mounts:      mounts,
		Environment: env,
		ClaudeArgs:  args,
		WorkDir:     "/workspace",
		User:        cfg.Container.User,
		MemoryLimit: cfg.Container.MemoryLimit,
		Network:     cfg.Container.Network,
		Security: container.SecurityOptions{
			DropCapabilities: cfg.Security.DropCapabilities,
			NoNewPrivileges:  cfg.Security.NoNewPrivileges,
			ReadOnlyRoot:     cfg.Security.ReadOnlyRoot,
			CACerts:          caCerts,
		},
	}

	// Create and run container
	runner, err := container.NewRunner()
	if err != nil {
		return fmt.Errorf("failed to create container runner: %w", err)
	}
	defer runner.Close()

	return runner.Run(ctx, cancel, opts)
}
