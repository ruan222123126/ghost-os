package externalagent

import (
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

type ExecutionPolicy struct {
	PermissionMode string
	ApprovalPolicy string
	Sandbox        string
}

func ResolveExecutionPolicy(permissionMode string) (ExecutionPolicy, error) {
	mode := strings.TrimSpace(permissionMode)
	if mode == "" {
		mode = bridgeconfig.DefaultExternalCodexPermissionMode
	}
	switch mode {
	case bridgeconfig.ExternalCodexPermissionReadOnly:
		return ExecutionPolicy{PermissionMode: mode, ApprovalPolicy: "never", Sandbox: "read-only"}, nil
	case bridgeconfig.ExternalCodexPermissionDefault:
		return ExecutionPolicy{PermissionMode: mode, ApprovalPolicy: "untrusted", Sandbox: "workspace-write"}, nil
	case bridgeconfig.ExternalCodexPermissionSafeYolo:
		return ExecutionPolicy{PermissionMode: mode, ApprovalPolicy: "on-failure", Sandbox: "workspace-write"}, nil
	case bridgeconfig.ExternalCodexPermissionYolo:
		return ExecutionPolicy{PermissionMode: mode, ApprovalPolicy: "never", Sandbox: "danger-full-access"}, nil
	default:
		return ExecutionPolicy{}, fmt.Errorf("unsupported external codex permission_mode: %q", mode)
	}
}

func ValidateApprovalDecision(raw string) (string, error) {
	decision := strings.TrimSpace(raw)
	switch decision {
	case DecisionApproved,
		DecisionApprovedForSession,
		DecisionDenied,
		DecisionAbort:
		return decision, nil
	default:
		return "", fmt.Errorf("unsupported external approval decision: %q", decision)
	}
}

func decisionToWire(decision string, legacy bool) any {
	if legacy {
		return decision
	}
	switch decision {
	case DecisionApproved:
		return "accept"
	case DecisionApprovedForSession:
		return "acceptForSession"
	case DecisionAbort:
		return "cancel"
	default:
		return "decline"
	}
}

func decisionToMCPResponse(decision string) map[string]any {
	switch decision {
	case DecisionApproved, DecisionApprovedForSession:
		return map[string]any{"action": "accept", "content": nil, "_meta": nil}
	case DecisionAbort:
		return map[string]any{"action": "cancel", "content": nil, "_meta": nil}
	default:
		return map[string]any{"action": "decline", "content": nil, "_meta": nil}
	}
}
