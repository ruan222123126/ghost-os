package llm

import (
	"context"
	"encoding/json"
	"strings"
)

type toolCallStreamState struct {
	index   int
	call    ToolCall
	args    strings.Builder
	started bool
	ended   bool
}

type toolCallStreamSet struct {
	states map[int]*toolCallStreamState
	order  []int
}

func newToolCallStreamSet() toolCallStreamSet {
	return toolCallStreamSet{
		states: make(map[int]*toolCallStreamState),
	}
}

func (s *toolCallStreamSet) ensure(index int) *toolCallStreamState {
	if state, ok := s.states[index]; ok {
		return state
	}
	state := &toolCallStreamState{index: index}
	s.states[index] = state
	s.order = append(s.order, index)
	return state
}

func (s *toolCallStreamSet) finalizeMessage(message *Message) {
	if len(s.order) == 0 {
		return
	}

	message.ToolCalls = make([]ToolCall, 0, len(s.order))
	for _, index := range s.order {
		state := s.states[index]
		if state == nil {
			continue
		}
		message.ToolCalls = append(message.ToolCalls, state.toolCall())
	}
}

func (s *toolCallStreamSet) endAll(ctx context.Context, sink LLMStreamSink) error {
	for _, index := range s.order {
		state := s.states[index]
		if state == nil {
			continue
		}
		if err := state.end(ctx, sink); err != nil {
			return err
		}
	}
	return nil
}

func (s *toolCallStreamSet) len() int {
	return len(s.order)
}

func (s *toolCallStreamState) setID(id string) {
	if trimmed := strings.TrimSpace(id); trimmed != "" {
		s.call.ID = trimmed
	}
}

func (s *toolCallStreamState) setName(name string) {
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		s.call.Name = trimmed
	}
}

func (s *toolCallStreamState) start(ctx context.Context, sink LLMStreamSink) error {
	if s.started {
		return nil
	}
	if err := sink.OnDelta(ctx, LLMDelta{
		Kind:          DeltaKindToolCallStart,
		ToolCallIndex: s.index,
		ToolCallID:    s.call.ID,
		ToolName:      s.call.Name,
	}); err != nil {
		return err
	}
	s.started = true
	return nil
}

func (s *toolCallStreamState) appendArguments(ctx context.Context, sink LLMStreamSink, fragment string) error {
	if fragment == "" {
		return nil
	}
	s.args.WriteString(fragment)
	return sink.OnDelta(ctx, LLMDelta{
		Kind:              DeltaKindToolCallDelta,
		ToolCallIndex:     s.index,
		ArgumentsFragment: fragment,
	})
}

func (s *toolCallStreamState) seedArguments(raw string) {
	if s.args.Len() == 0 && strings.TrimSpace(raw) != "" {
		s.args.WriteString(raw)
	}
}

func (s *toolCallStreamState) end(ctx context.Context, sink LLMStreamSink) error {
	if s.ended {
		return nil
	}
	if err := sink.OnDelta(ctx, LLMDelta{
		Kind:          DeltaKindToolCallEnd,
		ToolCallIndex: s.index,
	}); err != nil {
		return err
	}
	s.ended = true
	return nil
}

func (s *toolCallStreamState) toolCall() ToolCall {
	return ToolCall{
		ID:        s.call.ID,
		Name:      s.call.Name,
		Arguments: cloneRawJSON(json.RawMessage(s.args.String())),
	}
}
