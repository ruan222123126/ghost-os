package config

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	providerSyncStateFilename = "provider-sync.v1.json"
	providerSyncStateVersion  = 1
)

type providerSyncEntry struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name,omitempty"`
	UpdatedAt  string `json:"updated_at"`
	DeletedAt  string `json:"deleted_at,omitempty"`
}

type providerSyncState struct {
	Version   int                 `json:"version"`
	Providers []providerSyncEntry `json:"providers"`
}

func (s *store) listProvidersLocked(includeTombstones bool) ([]ProviderRecord, error) {
	fileCfg, configPath, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return nil, err
	}

	records := providerRecordsFromConfigs(normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model)))
	state, err := loadProviderSyncState(configPath)
	if err != nil {
		return nil, err
	}
	state, changed := reconcileProviderSyncState(state, records)
	if changed {
		if err := writeProviderSyncState(configPath, state); err != nil {
			return nil, err
		}
	}

	active := applyProviderSyncMetadata(records, state)
	if !includeTombstones {
		return active, nil
	}
	return append(active, tombstoneProviderRecords(state, active)...), nil
}

func loadProviderSyncState(configPath string) (providerSyncState, error) {
	path := providerSyncStatePath(configPath)
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
			return providerSyncState{Version: providerSyncStateVersion}, nil
		}
		var state providerSyncState
		if err := json.Unmarshal(raw, &state); err != nil {
			return providerSyncState{}, fmt.Errorf("decode provider sync state: %w", err)
		}
		return normalizeProviderSyncState(state), nil
	case os.IsNotExist(err):
		return providerSyncState{Version: providerSyncStateVersion}, nil
	default:
		return providerSyncState{}, fmt.Errorf("read provider sync state: %w", err)
	}
}

func writeProviderSyncState(configPath string, state providerSyncState) error {
	path := providerSyncStatePath(configPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create provider sync directory: %w", err)
	}
	encoded, err := json.MarshalIndent(normalizeProviderSyncState(state), "", "  ")
	if err != nil {
		return fmt.Errorf("encode provider sync state: %w", err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		return fmt.Errorf("write provider sync state: %w", err)
	}
	return nil
}

func providerSyncStatePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), providerSyncStateFilename)
}

func reconcileProviderSyncState(state providerSyncState, active []ProviderRecord) (providerSyncState, bool) {
	state = normalizeProviderSyncState(state)
	changed := false
	now := currentProviderSyncTimestamp()
	activeNames := make(map[string]struct{}, len(active))

	for _, record := range active {
		name := strings.TrimSpace(record.Name)
		if name == "" {
			continue
		}
		activeNames[strings.ToLower(name)] = struct{}{}
		index := findProviderSyncEntry(state.Providers, record.ProviderID, name)
		if index < 0 {
			state.Providers = append(state.Providers, providerSyncEntry{
				ProviderID: chooseProviderSyncID(record.ProviderID),
				Name:       name,
				UpdatedAt:  chooseProviderSyncTimestamp(record.UpdatedAt, now),
			})
			changed = true
			continue
		}

		entry := state.Providers[index]
		if strings.TrimSpace(entry.ProviderID) == "" {
			entry.ProviderID = chooseProviderSyncID(record.ProviderID)
			changed = true
		}
		if entry.Name != name {
			entry.Name = name
			changed = true
		}
		if strings.TrimSpace(entry.UpdatedAt) == "" {
			entry.UpdatedAt = chooseProviderSyncTimestamp(record.UpdatedAt, now)
			changed = true
		}
		if strings.TrimSpace(entry.DeletedAt) != "" {
			entry.DeletedAt = ""
			entry.UpdatedAt = chooseProviderSyncTimestamp(record.UpdatedAt, now)
			changed = true
		}
		state.Providers[index] = entry
	}

	for index, entry := range state.Providers {
		name := strings.ToLower(strings.TrimSpace(entry.Name))
		if name == "" {
			continue
		}
		if _, ok := activeNames[name]; ok {
			continue
		}
		if strings.TrimSpace(entry.DeletedAt) == "" {
			state.Providers[index].DeletedAt = chooseProviderSyncTimestamp(entry.UpdatedAt, now)
			if strings.TrimSpace(state.Providers[index].UpdatedAt) == "" {
				state.Providers[index].UpdatedAt = state.Providers[index].DeletedAt
			}
			changed = true
		}
	}

	return state, changed
}

func normalizeProviderSyncState(state providerSyncState) providerSyncState {
	state.Version = providerSyncStateVersion
	if len(state.Providers) == 0 {
		state.Providers = nil
		return state
	}

	out := make([]providerSyncEntry, 0, len(state.Providers))
	seen := make(map[string]struct{}, len(state.Providers))
	for _, entry := range state.Providers {
		entry.ProviderID = strings.TrimSpace(entry.ProviderID)
		entry.Name = strings.TrimSpace(entry.Name)
		entry.UpdatedAt = strings.TrimSpace(entry.UpdatedAt)
		entry.DeletedAt = strings.TrimSpace(entry.DeletedAt)
		key := providerSyncEntryKey(entry)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, entry)
	}
	if len(out) == 0 {
		state.Providers = nil
		return state
	}
	state.Providers = out
	return state
}

func applyProviderSyncMetadata(records []ProviderRecord, state providerSyncState) []ProviderRecord {
	if len(records) == 0 {
		return nil
	}

	out := make([]ProviderRecord, 0, len(records))
	for _, record := range records {
		entry, ok := findProviderSyncEntryValue(state.Providers, record.ProviderID, record.Name)
		if ok {
			record.ProviderID = entry.ProviderID
			record.UpdatedAt = entry.UpdatedAt
			record.DeletedAt = entry.DeletedAt
		}
		out = append(out, record)
	}
	return out
}

func tombstoneProviderRecords(state providerSyncState, active []ProviderRecord) []ProviderRecord {
	if len(state.Providers) == 0 {
		return nil
	}

	activeIDs := make(map[string]struct{}, len(active))
	for _, record := range active {
		if id := strings.TrimSpace(record.ProviderID); id != "" {
			activeIDs[id] = struct{}{}
		}
	}

	out := make([]ProviderRecord, 0, len(state.Providers))
	for _, entry := range state.Providers {
		if strings.TrimSpace(entry.DeletedAt) == "" {
			continue
		}
		if _, ok := activeIDs[strings.TrimSpace(entry.ProviderID)]; ok {
			continue
		}
		out = append(out, ProviderRecord{
			Name:       entry.Name,
			ProviderID: entry.ProviderID,
			UpdatedAt:  entry.UpdatedAt,
			DeletedAt:  entry.DeletedAt,
		})
	}
	return out
}

func upsertProviderSyncEntry(
	state providerSyncState,
	lookupName string,
	nextName string,
	providerID string,
	updatedAt string,
	deleted bool,
) providerSyncState {
	state = normalizeProviderSyncState(state)
	index := findProviderSyncEntry(state.Providers, providerID, lookupName)
	if index < 0 {
		index = findProviderSyncEntry(state.Providers, providerID, nextName)
	}
	name := strings.TrimSpace(nextName)
	timestamp := chooseProviderSyncTimestamp(updatedAt, currentProviderSyncTimestamp())
	if index < 0 {
		entry := providerSyncEntry{
			ProviderID: chooseProviderSyncID(providerID),
			Name:       name,
			UpdatedAt:  timestamp,
		}
		if deleted {
			entry.DeletedAt = timestamp
		}
		state.Providers = append(state.Providers, entry)
		return state
	}

	entry := state.Providers[index]
	if strings.TrimSpace(entry.ProviderID) == "" {
		entry.ProviderID = chooseProviderSyncID(providerID)
	}
	entry.Name = name
	entry.UpdatedAt = timestamp
	if deleted {
		entry.DeletedAt = timestamp
	} else {
		entry.DeletedAt = ""
	}
	state.Providers[index] = entry
	return state
}

func findProviderSyncEntry(entries []providerSyncEntry, providerID string, name string) int {
	if entry, ok := findProviderSyncEntryValue(entries, providerID, name); ok {
		for index, candidate := range entries {
			if candidate == entry {
				return index
			}
		}
	}
	return -1
}

func findProviderSyncEntryValue(entries []providerSyncEntry, providerID string, name string) (providerSyncEntry, bool) {
	id := strings.TrimSpace(providerID)
	if id != "" {
		for _, entry := range entries {
			if strings.TrimSpace(entry.ProviderID) == id {
				return entry, true
			}
		}
	}

	targetName := strings.ToLower(strings.TrimSpace(name))
	if targetName == "" {
		return providerSyncEntry{}, false
	}
	for _, entry := range entries {
		if strings.ToLower(strings.TrimSpace(entry.Name)) == targetName {
			return entry, true
		}
	}
	return providerSyncEntry{}, false
}

func providerSyncEntryKey(entry providerSyncEntry) string {
	if id := strings.TrimSpace(entry.ProviderID); id != "" {
		return "id:" + id
	}
	if name := strings.ToLower(strings.TrimSpace(entry.Name)); name != "" {
		return "name:" + name
	}
	return ""
}

func chooseProviderSyncID(raw string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return newProviderSyncID()
}

func chooseProviderSyncTimestamp(raw string, fallback string) string {
	if trimmed := strings.TrimSpace(raw); trimmed != "" {
		return trimmed
	}
	return fallback
}

func currentProviderSyncTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func newProviderSyncID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(fmt.Sprintf("generate provider sync id: %v", err))
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}
