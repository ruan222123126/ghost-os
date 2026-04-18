package session

import (
	"sort"
	"strings"
	"time"
)

type DynamicToolLoad struct {
	ToolName       string    `json:"tool_name"`
	LoadedAt       time.Time `json:"loaded_at,omitempty"`
	LoadedAtTurn   int       `json:"loaded_at_turn,omitempty"`
	LastCalledAt   time.Time `json:"last_called_at,omitempty"`
	LastCalledTurn int       `json:"last_called_turn,omitempty"`
	LoadedBy       string    `json:"loaded_by,omitempty"`
}

type DynamicToolLoadResult struct {
	Load          DynamicToolLoad
	AlreadyLoaded bool
}

func (s *Session) AdvanceToolTurn(idleTurns int) []string {
	if s == nil {
		return nil
	}

	s.TurnIndex++
	expired := s.pruneExpiredDynamicTools(idleTurns)
	s.pruneExpiredDynamicSkills(idleTurns)
	s.UpdatedAt = time.Now().UTC()
	return expired
}

func (s *Session) EnsureDynamicToolLoaded(toolName string, loadedBy string) DynamicToolLoadResult {
	if s == nil {
		return DynamicToolLoadResult{}
	}

	name := strings.TrimSpace(toolName)
	if name == "" {
		return DynamicToolLoadResult{}
	}
	if s.DynamicToolLoads == nil {
		s.DynamicToolLoads = make(map[string]DynamicToolLoad, 4)
	}

	now := time.Now().UTC()
	record, ok := s.DynamicToolLoads[name]
	if ok {
		record = normalizeDynamicToolLoad(name, record)
		s.DynamicToolLoads[name] = record
		return DynamicToolLoadResult{Load: record, AlreadyLoaded: true}
	}

	record = DynamicToolLoad{
		ToolName:     name,
		LoadedAt:     now,
		LoadedAtTurn: s.TurnIndex,
		LoadedBy:     strings.TrimSpace(loadedBy),
	}
	s.DynamicToolLoads[name] = record
	s.UpdatedAt = now
	return DynamicToolLoadResult{Load: record}
}

func (s *Session) UnloadDynamicTool(toolName string) bool {
	if s == nil || len(s.DynamicToolLoads) == 0 {
		return false
	}

	name := strings.TrimSpace(toolName)
	if name == "" {
		return false
	}
	if _, ok := s.DynamicToolLoads[name]; !ok {
		return false
	}

	delete(s.DynamicToolLoads, name)
	s.UpdatedAt = time.Now().UTC()
	return true
}

func (s *Session) NoteDynamicToolCall(toolName string) bool {
	if s == nil || len(s.DynamicToolLoads) == 0 {
		return false
	}

	name := strings.TrimSpace(toolName)
	record, ok := s.DynamicToolLoads[name]
	if !ok {
		return false
	}

	now := time.Now().UTC()
	record = normalizeDynamicToolLoad(name, record)
	record.LastCalledAt = now
	record.LastCalledTurn = s.TurnIndex
	s.DynamicToolLoads[name] = record
	s.UpdatedAt = now
	return true
}

func (s *Session) VisibleDynamicToolNames(idleTurns int) []string {
	if s == nil || len(s.DynamicToolLoads) == 0 {
		return nil
	}

	visible := make([]string, 0, len(s.DynamicToolLoads))
	for name, record := range s.DynamicToolLoads {
		record = normalizeDynamicToolLoad(name, record)
		if !record.VisibleForTurn(s.TurnIndex) || record.ExpiredAtTurn(s.TurnIndex, idleTurns) {
			continue
		}
		visible = append(visible, name)
	}
	sort.Strings(visible)
	return visible
}

func (s *Session) DynamicToolLoadsSnapshot() []DynamicToolLoad {
	if s == nil || len(s.DynamicToolLoads) == 0 {
		return nil
	}

	names := make([]string, 0, len(s.DynamicToolLoads))
	for name := range s.DynamicToolLoads {
		names = append(names, name)
	}
	sort.Strings(names)

	out := make([]DynamicToolLoad, 0, len(names))
	for _, name := range names {
		out = append(out, normalizeDynamicToolLoad(name, s.DynamicToolLoads[name]))
	}
	return out
}

func (s *Session) pruneExpiredDynamicTools(idleTurns int) []string {
	if len(s.DynamicToolLoads) == 0 {
		return nil
	}

	expired := make([]string, 0, len(s.DynamicToolLoads))
	for name, record := range s.DynamicToolLoads {
		record = normalizeDynamicToolLoad(name, record)
		if !record.ExpiredAtTurn(s.TurnIndex, idleTurns) {
			s.DynamicToolLoads[name] = record
			continue
		}
		delete(s.DynamicToolLoads, name)
		expired = append(expired, name)
	}
	if len(expired) == 0 {
		return nil
	}
	sort.Strings(expired)
	return expired
}

func normalizeDynamicToolLoad(toolName string, record DynamicToolLoad) DynamicToolLoad {
	normalizedName := strings.TrimSpace(toolName)
	if normalizedName == "" {
		normalizedName = strings.TrimSpace(record.ToolName)
	}
	record.ToolName = normalizedName
	record.LoadedBy = strings.TrimSpace(record.LoadedBy)
	return record
}

func (r DynamicToolLoad) VisibleForTurn(currentTurn int) bool {
	return r.LoadedAtTurn > 0 && r.LoadedAtTurn <= currentTurn
}

func (r DynamicToolLoad) ExpiredAtTurn(currentTurn int, idleTurns int) bool {
	if currentTurn <= 0 || idleTurns <= 0 {
		return false
	}
	return currentTurn-r.referenceTurn() > idleTurns
}

func (r DynamicToolLoad) RemainingIdleTurns(currentTurn int, idleTurns int) int {
	if idleTurns <= 0 {
		return 0
	}
	remaining := idleTurns - max(0, currentTurn-r.referenceTurn()-1)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r DynamicToolLoad) referenceTurn() int {
	if r.LastCalledTurn > r.LoadedAtTurn {
		return r.LastCalledTurn
	}
	return r.LoadedAtTurn
}
