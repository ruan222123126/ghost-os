package session

import "encoding/json"

const PendingComputerUseRunStateVersion = 1

type PendingComputerUseRunState struct {
	Version int             `json:"version,omitempty"`
	State   json.RawMessage `json:"state"`
}

func clonePendingComputerUseRunState(raw PendingComputerUseRunState) PendingComputerUseRunState {
	cloned := raw
	if len(raw.State) > 0 {
		cloned.State = append(json.RawMessage(nil), raw.State...)
	}
	return cloned
}
