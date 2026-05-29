package api

type SessionSourceAssignment struct {
	Kind      string `json:"kind"`
	OwnerID   string `json:"owner_id"`
	OwnerName string `json:"owner_name"`
}

type SessionSourceResolution struct {
	Assignments      map[string]SessionSourceAssignment `json:"assignments"`
	HiddenSessionIDs []string                           `json:"hidden_session_ids"`
}
