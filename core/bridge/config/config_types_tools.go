package config

type ToolRecord struct {
	Name            string
	Enabled         bool
	PromptOverride  string
	SandboxMemoryMB *int
}

type ToolUpdateRequest struct {
	Name            string
	Enabled         *bool
	PromptOverride  *string
	SandboxMemoryMB *int
}

type ToolSelectorConfig struct {
	Enabled    bool
	Mode       string
	Model      string
	TimeoutMS  int
	Confidence float64
	Shadow     bool
	RecentMsgs int
	// AllowlistOnly switches tool_allowlist from resident-only mode to strict static visibility mode.
	// When false, Allowlist defines resident tools and selector-visible static tools still include other non-blocked tools.
	AllowlistOnly   bool
	Allowlist       []string
	Blocklist       []string
	PromptOverrides map[string]string
}

type ToolSearchConfig struct {
	Enabled   bool
	IdleTurns int
}
