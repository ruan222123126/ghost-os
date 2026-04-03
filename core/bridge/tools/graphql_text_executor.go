package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"

	"ghost-os/bridge/llm"
)

type GraphQLTextExecutionResult struct {
	Recognized bool
	Calls      []GraphQLTextToolCall

	// Legacy single-call view retained for compatibility with existing callers.
	Operation ast.Operation
	ToolName  string
	Arguments json.RawMessage
}

type GraphQLTextToolCall struct {
	Operation ast.Operation
	ToolID    int
	ToolName  string
	Arguments json.RawMessage
}

type GraphQLTextExecutor interface {
	Execute(ctx context.Context, text string, traceID string) (GraphQLTextExecutionResult, error)
}

type GraphQLTextExecutorOptions struct {
	SanitizeKnownArtifacts bool
}

type graphQLTextExecutor struct {
	catalog                ToolCatalog
	sanitizeKnownArtifacts bool
}

func NewGraphQLTextExecutor(catalog ToolCatalog) GraphQLTextExecutor {
	return NewGraphQLTextExecutorWithOptions(catalog, GraphQLTextExecutorOptions{
		SanitizeKnownArtifacts: true,
	})
}

func NewGraphQLTextExecutorWithOptions(
	catalog ToolCatalog,
	opts GraphQLTextExecutorOptions,
) GraphQLTextExecutor {
	return &graphQLTextExecutor{
		catalog:                catalog,
		sanitizeKnownArtifacts: opts.SanitizeKnownArtifacts,
	}
}

func (e *graphQLTextExecutor) Execute(
	_ context.Context,
	text string,
	traceID string,
) (GraphQLTextExecutionResult, error) {
	normalized := normalizeGraphQLTextDocument(text, e.sanitizeKnownArtifacts)
	if !normalized.Recognized {
		return GraphQLTextExecutionResult{}, nil
	}
	logGraphQLTextSanitization(traceID, normalized.SanitizeKinds, e.sanitizeKnownArtifacts)
	if normalized.LegacyGraphQL {
		return GraphQLTextExecutionResult{Recognized: true}, newGraphQLTextParseError(
			fmt.Errorf("legacy GraphQL mutation/query tool calls are no longer supported"),
		)
	}

	call, err := parseGraphQLToolCallDocument(normalized.Document)
	if err != nil {
		call.Recognized = true
		return call, err
	}
	if e.catalog == nil {
		call.Recognized = true
		return call, newGraphQLTextCatalogUnavailableError()
	}

	entries := graphQLVisibleToolIDEntries(e.catalog)
	byID := make(map[int]llm.ToolDef, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry.Def
	}

	for index := range call.Calls {
		toolCall := &call.Calls[index]
		def, ok := byID[toolCall.ToolID]
		if !ok {
			call.Recognized = true
			return call, newGraphQLTextUnknownToolIDError(toolCall.ToolID, entries)
		}
		toolCall.ToolName = strings.TrimSpace(def.Name)
		if err := validateToolTagArguments(toolCall.Arguments, def); err != nil {
			call.Recognized = true
			return call, err
		}
	}

	call.Recognized = true
	if len(call.Calls) > 0 {
		call.ToolName = call.Calls[0].ToolName
	}
	return call, nil
}

func parseGraphQLToolCallDocument(text string) (GraphQLTextExecutionResult, error) {
	calls, recognized, err := parseToolTagCalls(text)
	if err != nil {
		return GraphQLTextExecutionResult{Recognized: recognized}, err
	}
	if !recognized {
		return GraphQLTextExecutionResult{}, nil
	}
	return graphQLTextExecutionResultForCalls(calls), nil
}

func parseToolTagCalls(text string) ([]GraphQLTextToolCall, bool, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, false, nil
	}

	const (
		modeNormal = iota
		modeCaptureID
		modeCaptureArgs
	)

	calls := make([]GraphQLTextToolCall, 0, 2)
	mode := modeNormal
	recognized := false
	tagStart := 0
	idStart := 0
	argsStart := 0
	currentToolID := 0
	inString := false
	escaped := false

	for index := 0; index < len(trimmed); {
		switch mode {
		case modeNormal:
			if strings.HasPrefix(trimmed[index:], "<t:") {
				recognized = true
				mode = modeCaptureID
				tagStart = index
				idStart = index + len("<t:")
				index = idStart
				continue
			}
			index++
		case modeCaptureID:
			if trimmed[index] != '>' {
				index++
				continue
			}
			toolID, err := parseToolTagID(trimmed[idStart:index])
			if err != nil {
				return nil, true, newGraphQLTextParseError(
					fmt.Errorf("invalid tool id near %q: %w", trimmed[tagStart:index+1], err),
				)
			}
			currentToolID = toolID
			mode = modeCaptureArgs
			argsStart = index + 1
			index = argsStart
			inString = false
			escaped = false
		case modeCaptureArgs:
			if !inString && strings.HasPrefix(trimmed[index:], "</t>") {
				arguments, err := decodeToolTagArguments(trimmed[argsStart:index])
				if err != nil {
					return nil, true, err
				}
				calls = append(calls, GraphQLTextToolCall{
					Operation: ast.Mutation,
					ToolID:    currentToolID,
					Arguments: arguments,
				})
				mode = modeNormal
				index += len("</t>")
				continue
			}
			ch := trimmed[index]
			if inString {
				if escaped {
					escaped = false
				} else {
					if ch == '\\' {
						escaped = true
					} else if ch == '"' {
						inString = false
					}
				}
			} else if ch == '"' {
				inString = true
			}
			index++
		}
	}

	if mode == modeCaptureID {
		return nil, true, newGraphQLTextParseError(fmt.Errorf("unterminated tool tag id"))
	}
	if mode == modeCaptureArgs {
		arguments, err := decodeToolTagArguments(trimmed[argsStart:])
		if err != nil {
			return nil, true, err
		}
		calls = append(calls, GraphQLTextToolCall{
			Operation: ast.Mutation,
			ToolID:    currentToolID,
			Arguments: arguments,
		})
	}
	if len(calls) == 0 {
		return nil, recognized, nil
	}
	return calls, true, nil
}

func parseToolTagID(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("tool id is empty")
	}
	for _, ch := range trimmed {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("tool id must be digits only")
		}
	}
	id, err := strconv.Atoi(trimmed)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("tool id must be a positive integer")
	}
	return id, nil
}

func decodeToolTagArguments(raw string) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return json.RawMessage(`{}`), nil
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil, newGraphQLTextParseError(fmt.Errorf("invalid tool arguments JSON: %w", err))
	}
	if _, ok := decoded.(map[string]any); !ok {
		return nil, newGraphQLTextParseError(fmt.Errorf("tool arguments must be a JSON object"))
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return nil, newGraphQLTextParseError(fmt.Errorf("normalize tool arguments JSON: %w", err))
	}
	return json.RawMessage(normalized), nil
}

func validateToolTagArguments(args json.RawMessage, def llm.ToolDef) error {
	var payload map[string]any
	if err := json.Unmarshal(args, &payload); err != nil {
		return newGraphQLTextArgumentError("", fmt.Errorf("decode arguments: %w", err))
	}
	if len(payload) == 0 {
		return nil
	}
	root := decodeGraphQLToolSchemaObject(def.Parameters)
	properties, _ := root["properties"].(map[string]any)
	for name, value := range payload {
		nested, ok := value.(map[string]any)
		if !ok {
			continue
		}
		if allowsNestedToolArgument(properties, name) {
			continue
		}
		if len(nested) == 0 {
			continue
		}
		return newGraphQLTextArgumentError(
			name,
			fmt.Errorf("argument %q should stay flat; nested objects are only allowed when the schema requires an object", name),
		)
	}
	return nil
}

func allowsNestedToolArgument(properties map[string]any, name string) bool {
	if len(properties) == 0 {
		return false
	}
	rawProperty, ok := properties[strings.TrimSpace(name)]
	if !ok {
		return false
	}
	property, _ := rawProperty.(map[string]any)
	typeName := strings.ToLower(strings.TrimSpace(graphQLSchemaTypeName(property)))
	if typeName == "object" {
		return true
	}
	if typeName != "" {
		return false
	}
	_, hasObjectFields := property["properties"].(map[string]any)
	return hasObjectFields
}

func graphQLTextExecutionResultForCalls(calls []GraphQLTextToolCall) GraphQLTextExecutionResult {
	result := GraphQLTextExecutionResult{
		Calls: append([]GraphQLTextToolCall(nil), calls...),
	}
	if len(calls) == 0 {
		return result
	}
	result.Operation = calls[0].Operation
	result.ToolName = calls[0].ToolName
	if len(calls[0].Arguments) != 0 {
		result.Arguments = append(json.RawMessage(nil), calls[0].Arguments...)
	}
	return result
}

func logGraphQLTextSanitization(traceID string, kinds []string, enabled bool) {
	if !enabled {
		return
	}
	trimmedTraceID := strings.TrimSpace(traceID)
	for _, kind := range kinds {
		if strings.TrimSpace(kind) == "" {
			continue
		}
		log.Printf(
			"trace_id=%s action=TAG_TOOL_TEXT_SANITIZE kind=%s",
			trimmedTraceID,
			strings.TrimSpace(kind),
		)
	}
}
