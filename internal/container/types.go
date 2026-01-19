package container

// Mount represents a bind mount configuration
type Mount struct {
	Source   string // Host path
	Target   string // Container path
	ReadOnly bool
}

// RunOptions configures container execution
type RunOptions struct {
	Image       string
	Mounts      []Mount
	Environment map[string]string
	ClaudeArgs  []string
	WorkDir     string
	User        string
	MemoryLimit string
	Network     string
	Security    SecurityOptions
}

// SecurityOptions configures container security settings
type SecurityOptions struct {
	DropCapabilities bool
	NoNewPrivileges  bool
	ReadOnlyRoot     bool
}

// BuildOptions configures image building
type BuildOptions struct {
	Dockerfile string
	ContextDir string
	Tag        string
	NoCache    bool
	Platform   string
}
