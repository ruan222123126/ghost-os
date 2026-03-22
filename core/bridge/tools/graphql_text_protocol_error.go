package tools

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type GraphQLTextProtocolFeedback struct {
	Status   string `json:"status,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Message  string `json:"message,omitempty"`
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
	}
	if line, column, ok := graphQLTextErrorLocation(err); ok {
		feedback.Line = line
		feedback.Column = column
	}
	return newGraphQLTextProtocolError(feedback, err, true)
}

func newGraphQLTextOperationCountError(received int) error {
	err := fmt.Errorf("graphql tool runtime requires exactly one operation")
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     "wrong_operation_count",
		Message:  err.Error(),
		Expected: "1",
		Received: strconv.Itoa(received),
	}, err, true)
}

func newGraphQLTextOperationError(kind string, message string, expected string, received string) error {
	err := fmt.Errorf("%s", strings.TrimSpace(message))
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     kind,
		Message:  err.Error(),
		Expected: expected,
		Received: received,
	}, err, true)
}

func newGraphQLTextFeatureError(kind string, message string) error {
	err := fmt.Errorf("%s", strings.TrimSpace(message))
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:    kind,
		Message: err.Error(),
	}, err, true)
}

func newGraphQLTextTopLevelFieldCountError(received int) error {
	err := fmt.Errorf("graphql tool runtime requires exactly one top-level field")
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     "wrong_top_level_field_count",
		Message:  err.Error(),
		Expected: "1",
		Received: strconv.Itoa(received),
	}, err, true)
}

func newGraphQLTextAliasError(fieldName string) error {
	err := fmt.Errorf("graphql tool runtime does not support aliases")
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:    "unsupported_alias",
		Message: err.Error(),
		Field:   strings.TrimSpace(fieldName),
	}, err, true)
}

func newGraphQLTextNestedSelectionError(fieldName string) error {
	err := fmt.Errorf("graphql tool runtime field %q does not support nested selections", fieldName)
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:    "unsupported_nested_selection",
		Message: err.Error(),
		Field:   strings.TrimSpace(fieldName),
	}, err, true)
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

func newGraphQLTextUnknownToolError(toolName string) error {
	err := fmt.Errorf("tool %q not found", strings.TrimSpace(toolName))
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:    "unknown_tool",
		Tool:    toolName,
		Message: err.Error(),
	}, err, true)
}

func newGraphQLTextWrongOperationError(
	toolName string,
	expected ast.Operation,
	received ast.Operation,
) error {
	err := fmt.Errorf("tool %q must use %s, got %s", strings.TrimSpace(toolName), expected, received)
	return newGraphQLTextProtocolError(GraphQLTextProtocolFeedback{
		Kind:     "wrong_operation",
		Tool:     toolName,
		Message:  err.Error(),
		Expected: string(expected),
		Received: string(received),
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
