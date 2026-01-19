# enclaude

> [!CAUTION]
> This solution is a work in progress and is not ready for consumption.

A secure, containerized sandbox for Claude Code with filesystem isolation.

## Overview

Enclaude runs Claude Code in an isolated Docker container, preventing access to sensitive host files while providing convenient credential passthrough. Your working directory and Claude settings are mounted automatically, but sensitive files like SSH keys, GPG keys, and cloud credentials are protected by default.

## Features

- **Filesystem Isolation**: Claude Code runs in a container with controlled access
- **Automatic Credential Passthrough**: Anthropic API key, GitHub, Google Cloud, and SSH credentials
- **Security Hardening**: Dropped capabilities, no-new-privileges, read-only root filesystem
- **Seamless Experience**: Interactive mode with working directory and `~/.claude` mounted
- **Customizable**: YAML configuration and custom Docker images

## Quick Start

```bash
# Install enclaude
go install github.com/jake-mok-nelson/enclaude/cmd/enclaude@latest

# Build the Docker image
enclaude build

# Run Claude Code in current directory
enclaude
```

## Installation

### From Source

```bash
git clone https://github.com/jake-mok-nelson/enclaude.git
cd enclaude
task build
task install
```

### Build the Docker Image

```bash
# Build the default image
enclaude build

# Or use task
task docker:build
```

## Usage

```bash
# Basic - interactive Claude Code in current directory
enclaude

# Mount additional directories
enclaude -m ~/projects/shared-lib
enclaude --mount-ro ~/docs/api-spec  # read-only

# Pass arguments to Claude Code
enclaude -- --help
enclaude -- --model claude-sonnet-4-20250514

# Override working directory
enclaude -w ~/projects/other-project

# Disable credential passthrough for this session
enclaude --no-credentials

# Use custom Docker image
enclaude --image my-custom-enclaude:latest
```

## Configuration

Create a config file at `~/.config/enclaude/config.yaml`:

```bash
# Generate default config
enclaude config init

# Show current config
enclaude config show
```

### Configuration Options

```yaml
# Image settings
image:
  name: enclaude:latest
  dockerfile: ""  # Path to custom Dockerfile

# Default mounts
mounts:
  defaults:
    - path: ~/projects/shared-utils
      readonly: true
  claude_dir: readwrite  # none | readonly | readwrite

# Credential passthrough
credentials:
  anthropic: auto    # auto | enabled | disabled
  github: auto       # auto | enabled | disabled
  gcloud: auto       # auto | enabled | disabled
  ssh:
    enabled: false   # Explicit opt-in
    keys:
      - ~/.ssh/id_ed25519
      - ~/.ssh/id_ed25519.pub
    known_hosts: true
    agent_forwarding: true

# Environment variables
environment:
  passthrough:
    - TERM
    - COLORTERM
    - EDITOR
  custom:
    DEBUG: "false"

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
```

## Credential Passthrough

| Credential | Method | Config Key |
|------------|--------|------------|
| Anthropic API | `ANTHROPIC_API_KEY` env var | `credentials.anthropic` |
| GitHub | `GH_TOKEN` env var or `~/.config/gh/hosts.yml` | `credentials.github` |
| Google Cloud | ADC file mount | `credentials.gcloud` |
| SSH Keys | Specific keys mounted read-only | `credentials.ssh` |

Each credential can be set to:
- `auto`: Detect and use if present (default)
- `enabled`: Always attempt to pass through
- `disabled`: Never pass through

### SSH Key Handling

SSH credentials require explicit opt-in for security:

```yaml
credentials:
  ssh:
    enabled: true
    keys:
      - ~/.ssh/id_ed25519
      - ~/.ssh/id_ed25519.pub
    known_hosts: true
    agent_forwarding: true
```

- Only specified keys are mounted (read-only)
- The entire `~/.ssh` directory is never exposed
- SSH agent forwarding via `SSH_AUTH_SOCK`

## Security

### Hardcoded Denied Paths

These paths are **always blocked** and cannot be overridden:
- `~/.gnupg` - GPG keys
- `~/.netrc` - Network credentials
- `~/.docker/config.json` - Docker credentials
- `~/.kube/config` - Kubernetes config
- `~/.aws/credentials` - AWS credentials

### Container Hardening

By default, enclaude applies these security measures:
- All Linux capabilities dropped
- `no-new-privileges` security option
- Read-only root filesystem
- Non-root user execution
- Memory limits

## Custom Images

Create custom images with additional tools:

```bash
# Build from custom Dockerfile
enclaude build -f docker/examples/Dockerfile.python -t enclaude-python:latest

# Use the custom image
enclaude --image enclaude-python:latest
```

Example Dockerfiles are provided in `docker/examples/`:
- `Dockerfile.python` - Python development environment
- `Dockerfile.go` - Go development environment

## Development

Requires [Task](https://taskfile.dev/) for build automation.

```bash
# Build
task build

# Run tests
task test

# Lint
task lint

# Build release binaries
task release

# Generate shell completions
task completions

# List all tasks
task --list
```

## Shell Completions

```bash
# Bash
source <(enclaude completion bash)

# Zsh
enclaude completion zsh > "${fpath[1]}/_enclaude"

# Fish
enclaude completion fish | source
```

## How It Works

1. When you run `enclaude`, it creates a Docker container with:
   - Your current directory mounted at `/workspace`
   - Your `~/.claude` directory mounted (configurable)
   - Environment variables for API credentials
   - Read-only mounts for file-based credentials

2. Claude Code runs interactively inside the container
3. When you quit Claude (or the session ends), the container is automatically removed
4. Files created in `/workspace` persist on your host system

## Troubleshooting

### "Image not found"
Build the image first:
```bash
enclaude build
```

### "Permission denied" on created files
Set the user mapping:
```bash
enclaude --user $(id -u):$(id -g)
```
Or in config:
```yaml
container:
  user: auto
```

### Credential not working
Check credential detection:
```bash
enclaude config show
```
Ensure the credential is set to `auto` or `enabled`.

## License

MIT License - see [LICENSE](LICENSE) for details.
