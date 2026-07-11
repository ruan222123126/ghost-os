package api

type WorkflowNodeContract struct {
	ID    string                    `json:"id"`
	Type  string                    `json:"type"`
	Start WorkflowStartNodeContract `json:"start,omitempty"`
	Tool  WorkflowToolNodeContract  `json:"tool,omitempty"`
	Llm   WorkflowLLMNodeContract   `json:"llm,omitempty"`
	Agent WorkflowAgentNodeContract `json:"agent,omitempty"`
	If    WorkflowIfNodeContract    `json:"if,omitempty"`
	Loop  WorkflowLoopNodeContract  `json:"loop,omitempty"`
}

type WorkflowEdgeContract struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
}

type WorkflowDefinitionContract struct {
	Nodes []WorkflowNodeContract `json:"nodes"`
	Edges []WorkflowEdgeContract `json:"edges"`
}

type OrchestrationNodeContract struct {
	ID    string                         `json:"id"`
	Type  string                         `json:"type"`
	Group OrchestrationGroupNodeContract `json:"group,omitempty"`
	Agent OrchestrationAgentNodeContract `json:"agent,omitempty"`
}

type OrchestrationGroupNodeContract struct {
	Title         string `json:"title"`
	SharedContext string `json:"shared_context"`
	SpeakingMode  string `json:"speaking_mode"`
	OwnerAgentID  string `json:"owner_agent_id,omitempty"`
	MaxRounds     int    `json:"max_rounds"`
}

type OrchestrationAgentNodeContract struct {
	Title            string                       `json:"title"`
	Message          string                       `json:"message"`
	RuntimeOverrides TaskRuntimeOverridesContract `json:"runtime_overrides,omitempty"`
}

type TaskRuntimeOverridesContract struct {
	ProviderName      string   `json:"provider_name,omitempty"`
	Model             string   `json:"model,omitempty"`
	SystemPrompt      string   `json:"system_prompt,omitempty"`
	PresetID          string   `json:"preset_id,omitempty"`
	ToolAllowlist     []string `json:"tool_allowlist,omitempty"`
	ToolAllowlistOnly *bool    `json:"tool_allowlist_only,omitempty"`
	MaxTurns          *int     `json:"max_turns,omitempty"`
}

type TaskRelayConfigContract struct {
	StopPolicy         string `json:"stop_policy"`
	MaxRounds          int    `json:"max_rounds,omitempty"`
	ExecutionTimeoutMs int    `json:"execution_timeout_ms,omitempty"`
}

type WorkflowToolNodeContract struct {
	ToolName  string         `json:"tool_name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type OrchestrationEdgeContract struct {
	FromNodeID string `json:"from_node_id"`
	ToNodeID   string `json:"to_node_id"`
	Kind       string `json:"kind"`
}

type WorkflowLLMNodeContract struct {
	Prompt       string `json:"prompt"`
	SystemPrompt string `json:"system_prompt,omitempty"`
}

type OrchestrationDefinitionContract struct {
	Nodes []OrchestrationNodeContract `json:"nodes"`
	Edges []OrchestrationEdgeContract `json:"edges"`
}

type WorkflowAgentNodeContract struct {
	Message          string                       `json:"message"`
	RuntimeOverrides TaskRuntimeOverridesContract `json:"runtime_overrides,omitempty"`
}

type WorkflowIfNodeContract struct {
	SourceNodeID string `json:"source_node_id,omitempty"`
	Operator     string `json:"operator"`
	Value        string `json:"value,omitempty"`
	TrueNodeID   string `json:"true_node_id"`
	FalseNodeID  string `json:"false_node_id"`
}

type WorkflowLoopNodeContract struct {
	MaxIterations int    `json:"max_iterations"`
	BodyNodeID    string `json:"body_node_id"`
	ExitNodeID    string `json:"exit_node_id"`
}

type WorkflowStartNodeContract struct {
	Inputs []WorkflowInputVariableContract `json:"inputs,omitempty"`
}

type WorkflowInputVariableContract struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required,omitempty"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}
