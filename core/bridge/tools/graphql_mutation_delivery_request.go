package tools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/session"
)

const (
	graphQLMutationIdempotencyModeHeader       = "header"
	graphQLMutationIdempotencyModeVariablePath = "variable_path"
)

type graphQLFrozenMutationRequest struct {
	DeliveryKey string
	Variables   map[string]any
	RequestHash string
}

type graphQLMutationRequestFingerprint struct {
	Source        string                            `json:"source"`
	Domain        string                            `json:"domain"`
	Policy        string                            `json:"policy"`
	RootMutation  string                            `json:"root_mutation"`
	Query         string                            `json:"query"`
	OperationName string                            `json:"operation_name,omitempty"`
	Variables     map[string]any                    `json:"variables,omitempty"`
	Idempotency   graphQLMutationIdempotencyPayload `json:"idempotency"`
}

type graphQLMutationIdempotencyPayload struct {
	Mode         string `json:"mode"`
	Header       string `json:"header,omitempty"`
	VariablePath string `json:"variable_path,omitempty"`
	DeliveryKey  string `json:"delivery_key"`
}

func freezeGraphQLMutationRequest(
	prepared graphQLPreparedMutation,
	args graphQLMutationArgs,
) (graphQLFrozenMutationRequest, error) {
	deliveryKey, err := newGraphQLMutationID("delivery")
	if err != nil {
		return graphQLFrozenMutationRequest{}, fmt.Errorf("generate delivery key: %w", err)
	}
	variables, err := injectGraphQLMutationDeliveryKey(
		cloneGraphQLMutationVariables(args.Variables),
		prepared.Policy.IdempotencyMode,
		prepared.Policy.IdempotencyVariablePath,
		deliveryKey,
	)
	if err != nil {
		return graphQLFrozenMutationRequest{}, err
	}
	requestHash, err := hashGraphQLMutationRequest(
		prepared,
		args.OperationName,
		variables,
		deliveryKey,
	)
	if err != nil {
		return graphQLFrozenMutationRequest{}, err
	}
	return graphQLFrozenMutationRequest{
		DeliveryKey: deliveryKey,
		Variables:   variables,
		RequestHash: requestHash,
	}, nil
}

func injectGraphQLMutationDeliveryKey(
	variables map[string]any,
	mode string,
	variablePath string,
	deliveryKey string,
) (map[string]any, error) {
	switch strings.TrimSpace(mode) {
	case graphQLMutationIdempotencyModeHeader:
		return variables, nil
	case graphQLMutationIdempotencyModeVariablePath:
		return setGraphQLMutationVariablePath(variables, variablePath, deliveryKey)
	default:
		return nil, fmt.Errorf("graphql mutation idempotency_mode %q is not supported", mode)
	}
}

func setGraphQLMutationVariablePath(
	variables map[string]any,
	path string,
	deliveryKey string,
) (map[string]any, error) {
	segments := graphQLMutationVariablePathSegments(path)
	if len(segments) == 0 {
		return nil, fmt.Errorf("graphql mutation idempotency_variable_path is required")
	}
	if len(segments) == 1 {
		return setGraphQLMutationVariableRoot(variables, segments[0], deliveryKey)
	}
	if len(variables) == 0 {
		return nil, fmt.Errorf(
			"graphql mutation idempotency_variable_path %q requires existing variables",
			path,
		)
	}
	target, err := resolveGraphQLMutationVariableParent(variables, segments)
	if err != nil {
		return nil, err
	}
	if err := setGraphQLMutationVariableValue(
		target,
		segments[len(segments)-1],
		deliveryKey,
	); err != nil {
		return nil, err
	}
	return variables, nil
}

func setGraphQLMutationVariableRoot(
	variables map[string]any,
	key string,
	deliveryKey string,
) (map[string]any, error) {
	if variables == nil {
		variables = make(map[string]any, 1)
	}
	if err := setGraphQLMutationVariableValue(variables, key, deliveryKey); err != nil {
		return nil, err
	}
	return variables, nil
}

func graphQLMutationVariablePathSegments(path string) []string {
	raw := strings.Split(strings.TrimSpace(path), ".")
	segments := make([]string, 0, len(raw))
	for _, segment := range raw {
		trimmed := strings.TrimSpace(segment)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}
	return segments
}

func resolveGraphQLMutationVariableParent(
	variables map[string]any,
	segments []string,
) (map[string]any, error) {
	current := variables
	for _, segment := range segments[:len(segments)-1] {
		next, err := graphQLMutationVariableObject(current, segment)
		if err != nil {
			return nil, err
		}
		current = next
	}
	return current, nil
}

func graphQLMutationVariableObject(
	current map[string]any,
	segment string,
) (map[string]any, error) {
	value, ok := current[segment]
	if !ok {
		return nil, fmt.Errorf(
			"graphql mutation idempotency_variable_path segment %q was not found",
			segment,
		)
	}
	next, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"graphql mutation idempotency_variable_path segment %q is not an object",
			segment,
		)
	}
	return next, nil
}

func setGraphQLMutationVariableValue(
	target map[string]any,
	key string,
	deliveryKey string,
) error {
	if existing, ok := target[key]; ok && existing != nil {
		if value, isString := existing.(string); isString && value == "" {
			target[key] = deliveryKey
			return nil
		}
		if existing != deliveryKey {
			return fmt.Errorf(
				"graphql mutation idempotency_variable_path %q already contains a conflicting value",
				key,
			)
		}
	}
	target[key] = deliveryKey
	return nil
}

func hashGraphQLMutationRequest(
	prepared graphQLPreparedMutation,
	operationName string,
	variables map[string]any,
	deliveryKey string,
) (string, error) {
	payload := graphQLMutationRequestFingerprint{
		Source:        prepared.Source.Name,
		Domain:        prepared.Domain,
		Policy:        prepared.Policy.Name,
		RootMutation:  prepared.Policy.RootMutation,
		Query:         prepared.MutationDoc,
		OperationName: strings.TrimSpace(operationName),
		Variables:     cloneGraphQLMutationVariables(variables),
		Idempotency: graphQLMutationIdempotencyPayload{
			Mode:         prepared.Policy.IdempotencyMode,
			Header:       prepared.Policy.IdempotencyHeader,
			VariablePath: prepared.Policy.IdempotencyVariablePath,
			DeliveryKey:  deliveryKey,
		},
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode graphql mutation hash payload: %w", err)
	}
	return hashGraphQLMutationBytes(encoded), nil
}

func hashGraphQLMutationBytes(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func injectGraphQLMutationRequestHeaders(
	headers map[string]string,
	intent session.PendingGraphQLMutationIntent,
) (map[string]string, error) {
	out := cloneGraphQLHeaders(headers)
	if out == nil {
		out = make(map[string]string, 1)
	}
	switch strings.TrimSpace(intent.IdempotencyMode) {
	case graphQLMutationIdempotencyModeHeader:
		header := strings.TrimSpace(intent.IdempotencyHeader)
		if header == "" {
			return nil, fmt.Errorf(
				"graphql mutation intent %q is missing idempotency_header",
				intent.IntentID,
			)
		}
		out[header] = intent.DeliveryKey
		return out, nil
	case graphQLMutationIdempotencyModeVariablePath:
		return out, nil
	default:
		return nil, fmt.Errorf(
			"graphql mutation intent %q has unsupported idempotency_mode %q",
			intent.IntentID,
			intent.IdempotencyMode,
		)
	}
}
