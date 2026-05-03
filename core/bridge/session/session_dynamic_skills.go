package session

import (
	"sort"
	"strings"
	"time"
)

type DynamicSkillLoad struct {
	SkillName      string    `json:"skill_name"`
	LoadedAt       time.Time `json:"loaded_at,omitempty"`
	LoadedAtTurn   int       `json:"loaded_at_turn,omitempty"`
	LastCalledAt   time.Time `json:"last_called_at,omitempty"`
	LastCalledTurn int       `json:"last_called_turn,omitempty"`
	LoadedBy       string    `json:"loaded_by,omitempty"`
}

type DynamicSkillLoadResult struct {
	Load          DynamicSkillLoad
	AlreadyLoaded bool
}

func (s *Session) EnsureDynamicSkillLoaded(skillName string, loadedBy string) DynamicSkillLoadResult {
	if s == nil {
		return DynamicSkillLoadResult{}
	}
	name := strings.TrimSpace(skillName)
	if name == "" {
		return DynamicSkillLoadResult{}
	}
	if s.DynamicSkillLoads == nil {
		s.DynamicSkillLoads = make(map[string]DynamicSkillLoad, 4)
	}

	now := time.Now().UTC()
	record, ok := s.DynamicSkillLoads[name]
	if ok {
		record = normalizeDynamicSkillLoad(name, record)
		s.DynamicSkillLoads[name] = record
		return DynamicSkillLoadResult{Load: record, AlreadyLoaded: true}
	}
	record = DynamicSkillLoad{
		SkillName:      name,
		LoadedAt:       now,
		LoadedAtTurn:   s.TurnIndex,
		LoadedBy:       strings.TrimSpace(loadedBy),
		LastCalledAt:   time.Time{},
		LastCalledTurn: 0,
	}
	s.DynamicSkillLoads[name] = record
	s.UpdatedAt = now
	return DynamicSkillLoadResult{Load: record}
}

func (s *Session) UnloadDynamicSkill(skillName string) bool {
	if s == nil || len(s.DynamicSkillLoads) == 0 {
		return false
	}
	name := strings.TrimSpace(skillName)
	if name == "" {
		return false
	}
	if _, ok := s.DynamicSkillLoads[name]; !ok {
		return false
	}
	delete(s.DynamicSkillLoads, name)
	s.UpdatedAt = time.Now().UTC()
	return true
}

func (s *Session) NoteDynamicSkillCall(skillName string) bool {
	if s == nil || len(s.DynamicSkillLoads) == 0 {
		return false
	}
	name := strings.TrimSpace(skillName)
	record, ok := s.DynamicSkillLoads[name]
	if !ok {
		return false
	}
	now := time.Now().UTC()
	record = normalizeDynamicSkillLoad(name, record)
	record.LastCalledAt = now
	record.LastCalledTurn = s.TurnIndex
	s.DynamicSkillLoads[name] = record
	s.UpdatedAt = now
	return true
}

func (s *Session) VisibleDynamicSkillNames(idleTurns int) []string {
	if s == nil || len(s.DynamicSkillLoads) == 0 {
		return nil
	}
	visible := make([]string, 0, len(s.DynamicSkillLoads))
	for name, record := range s.DynamicSkillLoads {
		record = normalizeDynamicSkillLoad(name, record)
		if !record.VisibleForTurn(s.TurnIndex) || record.ExpiredAtTurn(s.TurnIndex, idleTurns) {
			continue
		}
		visible = append(visible, name)
	}
	sort.Strings(visible)
	return visible
}

func (s *Session) DynamicSkillLoadsSnapshot() []DynamicSkillLoad {
	if s == nil || len(s.DynamicSkillLoads) == 0 {
		return nil
	}
	names := make([]string, 0, len(s.DynamicSkillLoads))
	for name := range s.DynamicSkillLoads {
		names = append(names, name)
	}
	sort.Strings(names)

	loads := make([]DynamicSkillLoad, 0, len(names))
	for _, name := range names {
		loads = append(loads, normalizeDynamicSkillLoad(name, s.DynamicSkillLoads[name]))
	}
	return loads
}

func (s *Session) PruneInvisibleDynamicSkills(visible []string) []string {
	if s == nil || len(s.DynamicSkillLoads) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(visible))
	for _, name := range visible {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			allowed[trimmed] = true
		}
	}
	removed := make([]string, 0, len(s.DynamicSkillLoads))
	for name := range s.DynamicSkillLoads {
		if allowed[name] {
			continue
		}
		delete(s.DynamicSkillLoads, name)
		removed = append(removed, name)
	}
	if len(removed) == 0 {
		return nil
	}
	sort.Strings(removed)
	s.UpdatedAt = time.Now().UTC()
	return removed
}

func (s *Session) pruneExpiredDynamicSkills(idleTurns int) []string {
	if len(s.DynamicSkillLoads) == 0 {
		return nil
	}
	expired := make([]string, 0, len(s.DynamicSkillLoads))
	for name, record := range s.DynamicSkillLoads {
		record = normalizeDynamicSkillLoad(name, record)
		if !record.ExpiredAtTurn(s.TurnIndex, idleTurns) {
			s.DynamicSkillLoads[name] = record
			continue
		}
		delete(s.DynamicSkillLoads, name)
		expired = append(expired, name)
	}
	if len(expired) == 0 {
		return nil
	}
	sort.Strings(expired)
	return expired
}

func normalizeDynamicSkillLoad(skillName string, record DynamicSkillLoad) DynamicSkillLoad {
	normalizedName := strings.TrimSpace(skillName)
	if normalizedName == "" {
		normalizedName = strings.TrimSpace(record.SkillName)
	}
	record.SkillName = normalizedName
	record.LoadedBy = strings.TrimSpace(record.LoadedBy)
	return record
}

func (r DynamicSkillLoad) VisibleForTurn(currentTurn int) bool {
	return r.LoadedAtTurn > 0 && r.LoadedAtTurn <= currentTurn
}

func (r DynamicSkillLoad) ExpiredAtTurn(currentTurn int, idleTurns int) bool {
	if currentTurn <= 0 || idleTurns <= 0 {
		return false
	}
	return currentTurn-r.referenceTurn() > idleTurns
}

func (r DynamicSkillLoad) RemainingIdleTurns(currentTurn int, idleTurns int) int {
	if idleTurns <= 0 {
		return 0
	}
	remaining := idleTurns - max(0, currentTurn-r.referenceTurn()-1)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (r DynamicSkillLoad) referenceTurn() int {
	if r.LastCalledTurn > r.LoadedAtTurn {
		return r.LastCalledTurn
	}
	return r.LoadedAtTurn
}
