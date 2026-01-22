package config

// Authentication modes
const (
	AuthAuto    = "auto"
	AuthSession = "session"
	AuthAPIKey  = "api-key"
)

// Credential settings
const (
	CredentialAuto     = "auto"
	CredentialEnabled  = "enabled"
	CredentialDisabled = "disabled"
)

// Session directory settings
const (
	SessionNone      = "none"
	SessionReadOnly  = "readonly"
	SessionReadWrite = "readwrite"
)

// Network modes
const (
	NetworkBridge = "bridge"
	NetworkHost   = "host"
	NetworkNone   = "none"
)

// User settings
const (
	UserAuto = "auto"
)
