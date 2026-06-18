package providers

import (
	"fmt"
	"strings"
)

func NewState(records []Record, activeProvider string, model string) State {
	cloned := cloneRecords(records)
	active := stringValue(NormalizedActiveProviderName(cloned, stringPointer(activeProvider)))
	return State{
		Records:        cloned,
		ActiveProvider: active,
		Model:          strings.TrimSpace(model),
	}
}

func Add(state State, record Record) (Patch, error) {
	state = normalizeState(state)
	normalized, err := ValidateRecord(record)
	if err != nil {
		return Patch{}, err
	}
	if IndexByName(state.Records, normalized.Name) >= 0 {
		return Patch{}, fmt.Errorf("%w: %s", ErrExists, normalized.Name)
	}

	state.Records = append(state.Records, normalized)
	if strings.TrimSpace(state.ActiveProvider) == "" {
		state.ActiveProvider = normalized.Name
	}
	return state.patch(nil), nil
}

func Update(state State, name string, record Record) (Patch, error) {
	state = normalizeState(state)
	target := strings.TrimSpace(name)
	if target == "" {
		return Patch{}, ErrNameRequired
	}

	normalized, err := ValidateRecord(record)
	if err != nil {
		return Patch{}, err
	}

	index := IndexByName(state.Records, target)
	if index < 0 {
		return Patch{}, fmt.Errorf("%w: %s", ErrNotFound, target)
	}
	if duplicate := IndexByName(state.Records, normalized.Name); duplicate >= 0 && duplicate != index {
		return Patch{}, fmt.Errorf("%w: %s", ErrExists, normalized.Name)
	}

	current := state.Records[index]
	if normalized.APIKey == nil {
		normalized.APIKey = cloneOptionalStringPointer(current.APIKey)
	}
	state.Records[index] = normalized
	if strings.EqualFold(strings.TrimSpace(state.ActiveProvider), current.Name) {
		state.ActiveProvider = normalized.Name
	}
	return state.patch(nil), nil
}

func Delete(state State, name string) (Patch, error) {
	state = normalizeState(state)
	target := strings.TrimSpace(name)
	if target == "" {
		return Patch{}, ErrNameRequired
	}

	index := IndexByName(state.Records, target)
	if index < 0 {
		return Patch{}, fmt.Errorf("%w: %s", ErrNotFound, target)
	}

	deletedName := state.Records[index].Name
	state.Records = append(state.Records[:index], state.Records[index+1:]...)
	if strings.EqualFold(strings.TrimSpace(state.ActiveProvider), deletedName) {
		state.ActiveProvider = stringValue(NormalizedActiveProviderName(state.Records))
	}
	return state.patch(nil), nil
}

func SetActive(state State, name string) (Patch, error) {
	state = normalizeState(state)
	target := strings.TrimSpace(name)
	if target == "" {
		return Patch{}, ErrNameRequired
	}

	index := IndexByName(state.Records, target)
	if index < 0 {
		return Patch{}, fmt.Errorf("%w: %s", ErrNotFound, target)
	}

	activeProvider := state.Records[index]
	state.ActiveProvider = activeProvider.Name
	var model *string
	if len(activeProvider.Models) > 0 {
		model = stringPointer(activeProvider.Models[0])
	}
	return state.patch(model), nil
}

func normalizeState(state State) State {
	records := cloneRecords(state.Records)
	active := stringValue(NormalizedActiveProviderName(records, stringPointer(state.ActiveProvider)))
	return State{
		Records:        records,
		ActiveProvider: active,
		Model:          strings.TrimSpace(state.Model),
	}
}

func (state State) patch(model *string) Patch {
	return Patch{
		Records:        cloneRecords(state.Records),
		ActiveProvider: NormalizedActiveProviderName(state.Records, stringPointer(state.ActiveProvider)),
		Model:          cloneOptionalStringPointer(model),
	}
}

func cloneRecords(records []Record) []Record {
	if len(records) == 0 {
		return nil
	}
	out := make([]Record, 0, len(records))
	for _, record := range records {
		out = append(out, Record{
			Name:                       record.Name,
			Type:                       record.Type,
			BaseURL:                    record.BaseURL,
			APIKey:                     cloneOptionalStringPointer(record.APIKey),
			Models:                     append([]string(nil), record.Models...),
			ContextWindowTokens:        record.ContextWindowTokens,
			ResponseReserveTokens:      record.ResponseReserveTokens,
			ModelContextWindowTokens:   CloneModelTokenOverrides(record.ModelContextWindowTokens),
			ModelResponseReserveTokens: CloneModelTokenOverrides(record.ModelResponseReserveTokens),
		})
	}
	return out
}
