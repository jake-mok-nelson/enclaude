package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HardcodedDeniedPaths are ALWAYS blocked and cannot be overridden
var HardcodedDeniedPaths = []string{
	"~/.gnupg",
	"~/.netrc",
	"~/.docker/config.json",
	"~/.kube/config",
	"~/.aws/credentials",
}

// CredentialControlledPaths are blocked unless explicitly configured
// These are handled by the credentials package
var CredentialControlledPaths = []string{
	"~/.ssh",
	"~/.aws",
	"~/.config/gh",
	"~/.config/gcloud",
}

// ExpandPath expands ~ to the user's home directory and cleans the path
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}

	// Expand ~
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		path = home
	}

	// Clean and make absolute
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		path = filepath.Join(cwd, path)
	}

	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		// Path may not exist yet, which is okay
		if os.IsNotExist(err) {
			return path, nil
		}
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	return resolved, nil
}

// ValidateMountPath checks if a path is allowed to be mounted
func ValidateMountPath(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check against hardcoded denied paths
	for _, denied := range HardcodedDeniedPaths {
		deniedExpanded := expandTilde(denied, home)
		if pathMatches(path, deniedExpanded) {
			return fmt.Errorf("path is in hardcoded denied list: %s", denied)
		}
	}

	// Note: Credential-controlled paths are validated separately
	// by the credentials package when the credential is enabled

	return nil
}

// ValidateMountPathStrict checks against both hardcoded and credential-controlled paths
// Use this for user-provided mounts that aren't going through credential handling
func ValidateMountPathStrict(path string) error {
	if err := ValidateMountPath(path); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Check against credential-controlled paths
	for _, controlled := range CredentialControlledPaths {
		controlledExpanded := expandTilde(controlled, home)
		if pathMatches(path, controlledExpanded) {
			return fmt.Errorf("path requires explicit credential configuration: %s", controlled)
		}
	}

	return nil
}

// pathMatches checks if path is equal to or a child of target
func pathMatches(path, target string) bool {
	// Exact match
	if path == target {
		return true
	}

	// Check if path is a child of target using filepath.Rel
	rel, err := filepath.Rel(target, path)
	if err != nil {
		return false
	}
	// If relative path starts with "..", path is not under target
	return !strings.HasPrefix(rel, "..")
}

func expandTilde(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		return home
	}
	return path
}

// IsPathInDirectory checks if path is inside directory
func IsPathInDirectory(path, directory string) bool {
	return pathMatches(path, directory)
}

// PathExists checks if a path exists and matches the expected type.
// If expectDir is true, checks for directory; if false, checks for file.
func PathExists(path string, expectDir bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir() == expectDir
}

// FileExists checks if a path exists and is a file (not a directory).
func FileExists(path string) bool {
	return PathExists(path, false)
}

// DirExists checks if a path exists and is a directory.
func DirExists(path string) bool {
	return PathExists(path, true)
}
