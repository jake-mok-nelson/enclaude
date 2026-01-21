package container

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types"
	containerTypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-units"
	"github.com/moby/term"
)

// Runner manages Docker container operations
type Runner struct {
	client *client.Client
}

// NewRunner creates a new container runner
func NewRunner() (*Runner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify connection
	if _, err := cli.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to connect to Docker: %w", err)
	}

	return &Runner{client: cli}, nil
}

// Close closes the Docker client
func (r *Runner) Close() error {
	return r.client.Close()
}

// Run creates and runs a container with the given options
func (r *Runner) Run(ctx context.Context, cancel context.CancelFunc, opts RunOptions) error {
	// Build environment variables
	var env []string
	for k, v := range opts.Environment {
		env = append(env, k+"="+v)
	}

	// Ensure PATH includes Claude's install location
	env = append(env, "PATH=/usr/local/bin:/usr/bin:/bin")

	// Set HOME to a writable location when running as non-root user
	// This is needed because Claude Code writes to ~/.claude
	env = append(env, "HOME=/tmp")

	

	// Build command - just pass the args since the Dockerfile has ENTRYPOINT set to claude
	cmd := strslice.StrSlice{}
	cmd = append(cmd, opts.ClaudeArgs...)

	// Build mounts
	var mounts []mount.Mount
	for _, m := range opts.Mounts {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   m.Source,
			Target:   m.Target,
			ReadOnly: m.ReadOnly,
		})
	}

	// Add tmpfs mounts for writable areas when using read-only root
	if opts.Security.ReadOnlyRoot {
		tmpfsMounts := []string{"/tmp", "/run", "/var/tmp"}
		for _, path := range tmpfsMounts {
			mounts = append(mounts, mount.Mount{
				Type:   mount.TypeTmpfs,
				Target: path,
			})
		}
	}

	// Mount CA certificates if configured
	if len(opts.Security.CACerts) > 0 {
		for _, certPath := range opts.Security.CACerts {
			certName := filepath.Base(certPath)
			mounts = append(mounts, mount.Mount{
				Type:     mount.TypeBind,
				Source:   certPath,
				Target:   "/usr/local/share/ca-certificates/" + certName,
				ReadOnly: true,
			})
		}
		// Set NODE_EXTRA_CA_CERTS for Node.js applications (Claude uses Node.js)
		// NODE_EXTRA_CA_CERTS only accepts a single file path, so we only set it
		// when exactly one CA certificate is configured. For multiple certificates,
		// users should bundle them into a single PEM file.
		if len(opts.Security.CACerts) == 1 {
			certName := filepath.Base(opts.Security.CACerts[0])
			env = append(env, "NODE_EXTRA_CA_CERTS=/usr/local/share/ca-certificates/"+certName)
		}
	}

	// Determine user
	user := ""
	if opts.User == "auto" {
		user = fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())
	} else if opts.User != "" {
		user = opts.User
	}

	// Parse memory limit
	var memoryLimit int64
	if opts.MemoryLimit != "" {
		limit, err := units.RAMInBytes(opts.MemoryLimit)
		if err != nil {
			return fmt.Errorf("invalid memory limit %q: %w", opts.MemoryLimit, err)
		}
		memoryLimit = limit
	}

	// Determine if we should use TTY mode
	isTTY := term.IsTerminal(os.Stdin.Fd())

	// Container configuration
	// For non-TTY mode, don't attach stdout/stderr - use ContainerLogs instead
	containerConfig := &containerTypes.Config{
		Image:        opts.Image,
		Cmd:          cmd,
		Env:          env,
		WorkingDir:   opts.WorkDir,
		User:         user,
		Tty:          isTTY,
		OpenStdin:    true,
		AttachStdin:  true,
		AttachStdout: isTTY,
		AttachStderr: isTTY,
	}

	// Host configuration
	hostConfig := &containerTypes.HostConfig{
		Mounts:         mounts,
		NetworkMode:    containerTypes.NetworkMode(opts.Network),
		ReadonlyRootfs: opts.Security.ReadOnlyRoot,
		AutoRemove:     false, // Disabled - we clean up manually in defer
		Resources: containerTypes.Resources{
			Memory: memoryLimit,
		},
	}

	// Security settings
	if opts.Security.DropCapabilities {
		hostConfig.CapDrop = strslice.StrSlice{"ALL"}
	}

	if opts.Security.NoNewPrivileges {
		hostConfig.SecurityOpt = append(hostConfig.SecurityOpt, "no-new-privileges")
	}

	// Create the container
	resp, err := r.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, "")
	if err != nil {
		// Check if image needs to be pulled
		if strings.Contains(err.Error(), "No such image") {
			return fmt.Errorf("image %q not found; run 'enclaude build' first or pull the image", opts.Image)
		}
		return fmt.Errorf("failed to create container: %w", err)
	}
	containerID := resp.ID

	// Ensure cleanup
	defer func() {
		// Container should auto-remove, but force cleanup if needed
		_ = r.client.ContainerRemove(context.Background(), containerID, containerTypes.RemoveOptions{
			Force: true,
		})
	}()

	// Attach to container (stdin always, stdout/stderr only for TTY)
	attachOpts := containerTypes.AttachOptions{
		Stream: true,
		Stdin:  true,
		Stdout: isTTY,
		Stderr: isTTY,
	}

	attachResp, err := r.client.ContainerAttach(ctx, containerID, attachOpts)
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}
	defer attachResp.Close()

	// Start output goroutine for TTY mode (reads from attach)
	outputDone := make(chan error, 1)
	if isTTY {
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := attachResp.Reader.Read(buf)
				if n > 0 {
					os.Stdout.Write(buf[:n])
					os.Stdout.Sync()
				}
				if err != nil {
					outputDone <- err
					return
				}
			}
		}()
	}

	// Start the container
	if err := r.client.ContainerStart(ctx, containerID, containerTypes.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// For non-TTY mode, use ContainerLogs (output goes to Docker's log driver)
	if !isTTY {
		go func() {
			logs, err := r.client.ContainerLogs(ctx, containerID, containerTypes.LogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Follow:     true,
			})
			if err != nil {
				outputDone <- err
				return
			}
			defer logs.Close()
			_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
			outputDone <- err
		}()
	}

	// Set up TTY after output goroutine is reading
	var oldState *term.State
	if isTTY {
		r.resizeTty(ctx, containerID)

		oldState, err = term.SetRawTerminal(os.Stdin.Fd())
		if err != nil {
			return fmt.Errorf("failed to set raw terminal: %w", err)
		}
		defer term.RestoreTerminal(os.Stdin.Fd(), oldState)

		// Handle terminal resize signals
		go r.monitorTtySize(ctx, containerID)
	}

	// Copy stdin to container with Ctrl+C detection
	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				break
			}
			// Check for Ctrl+C (byte 0x03) in raw mode
			if isTTY && cancel != nil {
				for i := 0; i < n; i++ {
					if buf[i] == 0x03 {
						cancel()
						return
					}
				}
			}
			if _, err := attachResp.Conn.Write(buf[:n]); err != nil {
				break
			}
		}
		attachResp.CloseWrite()
	}()

	// Wait for container to exit
	statusCh, errCh := r.client.ContainerWait(ctx, containerID, containerTypes.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		<-outputDone // Always wait for output to complete
		if err != nil && ctx.Err() == nil {
			return fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		<-outputDone // Wait for output to complete
		if status.StatusCode != 0 {
			return fmt.Errorf("container exited with code %d", status.StatusCode)
		}
	case <-ctx.Done():
		// Context cancelled (Ctrl+C or signal), stop the container
		stopCtx := context.Background()
		timeout := 5
		_ = r.client.ContainerStop(stopCtx, containerID, containerTypes.StopOptions{Timeout: &timeout})
		return ctx.Err()
	}

	return nil
}

// resizeTty resizes the container TTY to match the current terminal size
func (r *Runner) resizeTty(ctx context.Context, containerID string) {
	winsize, err := term.GetWinsize(os.Stdout.Fd())
	if err != nil {
		return
	}
	r.client.ContainerResize(ctx, containerID, containerTypes.ResizeOptions{
		Height: uint(winsize.Height),
		Width:  uint(winsize.Width),
	})
}

// monitorTtySize monitors terminal size changes and resizes the container TTY
func (r *Runner) monitorTtySize(ctx context.Context, containerID string) {
	// Monitor for SIGWINCH signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-sigCh:
			r.resizeTty(ctx, containerID)
		case <-ctx.Done():
			return
		}
	}
}

// Build builds a Docker image from a Dockerfile
func (r *Runner) Build(ctx context.Context, opts BuildOptions) error {
	// Read the Dockerfile
	dockerfileContent, err := os.ReadFile(opts.Dockerfile)
	if err != nil {
		return fmt.Errorf("failed to read Dockerfile: %w", err)
	}

	// Create a tar archive of the build context
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	// Add Dockerfile to the tar
	dockerfileHeader := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfileContent)),
	}
	if err := tw.WriteHeader(dockerfileHeader); err != nil {
		return fmt.Errorf("failed to write Dockerfile header: %w", err)
	}
	if _, err := tw.Write(dockerfileContent); err != nil {
		return fmt.Errorf("failed to write Dockerfile: %w", err)
	}

	// Walk the context directory and add files
	if err := filepath.Walk(opts.ContextDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the Dockerfile since we already added it
		if filepath.Base(path) == "Dockerfile" && filepath.Dir(path) == opts.ContextDir {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(opts.ContextDir, path)
		if err != nil {
			return err
		}

		// Skip hidden files/dirs except .dockerignore
		if strings.HasPrefix(filepath.Base(path), ".") && filepath.Base(path) != ".dockerignore" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// Write file content if not a directory
		if !info.IsDir() {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(content); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}

	if err := tw.Close(); err != nil {
		return fmt.Errorf("failed to close tar writer: %w", err)
	}

	// Build options
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{opts.Tag},
		NoCache:    opts.NoCache,
		Remove:     true,
	}

	if opts.Platform != "" {
		buildOptions.Platform = opts.Platform
	}

	// Build the image
	resp, err := r.client.ImageBuild(ctx, buf, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	// Stream build output
	_, err = io.Copy(os.Stdout, resp.Body)
	if err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	return nil
}

// ImageExists checks if an image exists locally
func (r *Runner) ImageExists(ctx context.Context, image string) (bool, error) {
	_, _, err := r.client.ImageInspectWithRaw(ctx, image)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// parseMemoryLimit parses memory limit strings like "4g", "512m"
func parseMemoryLimit(limit string) (int64, error) {
	limit = strings.ToLower(strings.TrimSpace(limit))
	if limit == "" {
		return 0, nil
	}

	multiplier := int64(1)
	if strings.HasSuffix(limit, "g") {
		multiplier = 1024 * 1024 * 1024
		limit = strings.TrimSuffix(limit, "g")
	} else if strings.HasSuffix(limit, "m") {
		multiplier = 1024 * 1024
		limit = strings.TrimSuffix(limit, "m")
	} else if strings.HasSuffix(limit, "k") {
		multiplier = 1024
		limit = strings.TrimSuffix(limit, "k")
	}

	value, err := strconv.ParseInt(limit, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory limit: %w", err)
	}

	return value * multiplier, nil
}
