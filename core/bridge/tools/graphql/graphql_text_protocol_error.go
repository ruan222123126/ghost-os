package graphql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

const graphQLToolSearchToolName = "tfind"

type GraphQLTextProtocolFeedback struct {
	Status   string `json:"status,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Message  string `json:"message,omitempty"`
	Hint     string `json:"hint,omitempty"`
	Example  string `json:"example,omitempty"`
	Expected string `json:"expected,omitempty"`
	Received string `json:"received,omitempty"`
	Field    string `json:"field,omitempty"`
	Argument string `json:"argument,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
}

type GraphQLTextProtocolError struct {
	feedback    GraphQLTextProtocolFeedback
	cause       error
	recoverable bool
}

func (e *GraphQLTextProtocolError) Error() string {
	if e == nil {
		return ""
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return e.feedback.Message
}

func (e *GraphQLTextProtocolError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *GraphQLTextProtocolError) Feedback() GraphQLTextProtocolFeedback {
	if e == nil {
		return GraphQLTextProtocolFeedback{}
	}
	return e.feedback
}

func (e *GraphQLTextProtocolError) Recoverable() bool {
	return e != nil && e.recoverable
}

func AsGraphQLTextProtocolError(err error) (*GraphQLTextProtocolError, bool) {
	var protocolErr *GraphQLTextProtocolError
	if !errors.As(err, &protocolErr) || protocolErr == nil {
		return nil, false
	}
	return protocolErr, true
}

func newGraphQLTextProtocolError(
	feedback GraphQLTextProtocolFeedback,
	cause error,
	recoverable bool,
) error {
	feedback.Status = "error"
	feedback.Kind = strings.TrimSpace(feedback.Kind)
	feedback.Tool = strings.TrimSpace(feedback.Tool)
	feedback.Message = strings.TrimSpace(feedback.Message)
	feedback.Hint = strings.TrimSpace(feedback.Hint)
	feedback.Example = strings.TrimSpace(feedback.Example)
	feedback.Expected = strings.TrimSpace(feedback.Expected)
	feedback.Received = strings.TrimSpace(feedback.Received)
	feedback.Field = strings.TrimSpace(feedback.Field)
	feedback.Argument = strings.TrimSpace(feedback.Argument)
	if feedback.Message == "" && cause != nil {
		feedback.Message = strings.TrimSpace(cause.Error())
	}
	return &GraphQLTextProtocolError{
		feedback:    feedback,
		cause:       cause,
		recoverable: recoverable,
	}
}

func newGraphQLTextParseError(err error) error {
	feedback := GraphQLTextProtocolFeedback{
		Kind:    "parse_error",
		Message: strings.TrimSpace(err.Error()),
		Hint:    "Return tool calls only with <t:TOOL_ID>JSON_ARGS</t>. Keep JSON args as a single object and do not include extra wrappers.",
		Example: `<t:1>{"query":"OpenAI API docs"}</t>`,
	}
	if line, column, ok := graphQLTextErrorLocation(err); ok {
		feedback.Line = line
		feedback.Column = column
	}
	return newGraphQLTextProtocolError(feedback, err, true)
}

func newGraphQLTextArgumentError(argumentName string, err error) error {
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     "invalid_argument",
		Message:  strings.TrimSpace(err.Error()),
		Argument: strings.TrimSpace(argumentName),
	}, err, true)
}

func newGraphQLTextCatalogUnavailableError() error {
	err := fmt.Errorf("graphql tool runtime catalog is not configured")
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:    "catalog_unavailable",
		Message: err.Error(),
	}, err, false)
}

func newGraphQLTextUnknownToolIDError(toolID int, entries []GraphQLToolIDEntry) error {
	err := fmt.Errorf("tool id %d not found", toolID)
	message := err.Error()
	if visible := graphQLVisibleToolIDListText(entries); visible != "" {
		message = fmt.Sprintf("%s; visible tools: %s", message, visible)
	}
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     "unknown_tool_id",
		Tool:     strconv.Itoa(toolID),
		Message:  message,
		Hint:     "Use one of the visible numeric tool IDs and call it as <t:ID>{...}</t>.",
		Example:  graphQLUnknownToolIDExample(entries),
		Expected: graphQLVisibleToolIDListText(entries),
		Received: strconv.Itoa(toolID),
	}, err, true)
}

func graphQLTextErrorLocation(err error) (int, int, bool) {
	var parseList gqlerror.List
	if errors.As(err, &parseList) {
		for _, item := range parseList {
			if item == nil || len(item.Locations) == 0 {
				continue
			}
			location := item.Locations[0]
			return location.Line, location.Column, true
		}
	}

	var parseErr *gqlerror.Error
	if errors.As(err, &parseErr) && parseErr != nil && len(parseErr.Locations) != 0 {
		location := parseErr.Locations[0]
		return location.Line, location.Column, true
	}
	return 0, 0, false
}

func graphQLUnknownToolIDExample(entries []GraphQLToolIDEntry) string {
	for _, entry := range entries {
		if strings.TrimSpace(entry.Def.Name) == graphQLToolSearchToolName {
			return fmt.Sprintf(`<t:%d>{"action":"search","query":"browser control"}</t>`, entry.ID)
		}
	}
	if len(entries) == 0 {
		return ""
	}
	return fmt.Sprintf(`<t:%d>{}</t>`, entries[0].ID)
}

func graphQLVisibleToolIDListText(entries []GraphQLToolIDEntry) string {
	if len(entries) == 0 {
		return ""
	}
	items := make([]string, 0, len(entries))
	for _, entry := range entries {
		toolName := strings.TrimSpace(entry.Def.Name)
		if toolName == "" {
			continue
		}
		items = append(items, fmt.Sprintf("%d:%s", entry.ID, toolName))
	}
	return strings.Join(items, ", ")
}
