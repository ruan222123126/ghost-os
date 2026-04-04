package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

const (
	tagParserModeNormal = iota
	tagParserModeCaptureID
	tagParserModeCaptureArgs
)

type toolTagParser struct {
	text          string
	index         int
	mode          int
	recognized    bool
	tagStart      int
	idStart       int
	argsStart     int
	currentToolID int
	inString      bool
	escaped       bool
	calls         []GraphQLTextToolCall
}

func parseToolTagCalls(text string) ([]GraphQLTextToolCall, bool, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, false, nil
	}
	parser := toolTagParser{
		text:  trimmed,
		mode:  tagParserModeNormal,
		calls: make([]GraphQLTextToolCall, 0, 2),
	}
	for parser.index < len(parser.text) {
		if err := parser.step(); err != nil {
			return nil, parser.recognized, err
		}
	}
	if err := parser.finish(); err != nil {
		return nil, parser.recognized, err
	}
	if len(parser.calls) == 0 {
		return nil, parser.recognized, nil
	}
	return parser.calls, true, nil
}

func (p *toolTagParser) step() error {
	switch p.mode {
	case tagParserModeNormal:
		p.consumeNormal()
		return nil
	case tagParserModeCaptureID:
		return p.consumeToolID()
	case tagParserModeCaptureArgs:
		return p.consumeToolArguments()
	default:
		return nil
	}
}

func (p *toolTagParser) consumeNormal() {
	if strings.HasPrefix(p.text[p.index:], "<t:") {
		p.recognized = true
		p.mode = tagParserModeCaptureID
		p.tagStart = p.index
		p.idStart = p.index + len("<t:")
		p.index = p.idStart
		return
	}
	p.index++
}

func (p *toolTagParser) consumeToolID() error {
	if p.text[p.index] != '>' {
		p.index++
		return nil
	}
	toolID, err := parseToolTagID(p.text[p.idStart:p.index])
	if err != nil {
		return newGraphQLTextParseError(
			fmt.Errorf("invalid tool id near %q: %w", p.text[p.tagStart:p.index+1], err),
		)
	}
	p.currentToolID = toolID
	p.mode = tagParserModeCaptureArgs
	p.argsStart = p.index + 1
	p.index = p.argsStart
	p.inString = false
	p.escaped = false
	return nil
}

func (p *toolTagParser) consumeToolArguments() error {
	if !p.inString && strings.HasPrefix(p.text[p.index:], "</t>") {
		if err := p.appendCurrentCall(p.text[p.argsStart:p.index]); err != nil {
			return err
		}
		p.mode = tagParserModeNormal
		p.index += len("</t>")
		return nil
	}
	p.consumeJSONChar(p.text[p.index])
	p.index++
	return nil
}

func (p *toolTagParser) consumeJSONChar(ch byte) {
	if p.inString {
		if p.escaped {
			p.escaped = false
			return
		}
		if ch == '\\' {
			p.escaped = true
			return
		}
		if ch == '"' {
			p.inString = false
		}
		return
	}
	if ch == '"' {
		p.inString = true
	}
}

func (p *toolTagParser) finish() error {
	switch p.mode {
	case tagParserModeCaptureID:
		return newGraphQLTextParseError(fmt.Errorf("unterminated tool tag id"))
	case tagParserModeCaptureArgs:
		return p.appendCurrentCall(p.text[p.argsStart:])
	default:
		return nil
	}
}

func (p *toolTagParser) appendCurrentCall(rawArgs string) error {
	arguments, err := decodeToolTagArguments(rawArgs)
	if err != nil {
		return err
	}
	p.calls = append(p.calls, GraphQLTextToolCall{
		Operation: ast.Mutation,
		ToolID:    p.currentToolID,
		Arguments: arguments,
	})
	return nil
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
