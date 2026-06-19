package api

type SessionSidebarPartition struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SessionSidebarPartitionState struct {
	Version     int                       `json:"version"`
	Partitions  []SessionSidebarPartition `json:"partitions"`
	Assignments map[string]string         `json:"assignments"`
}

type SessionSidebarPartitionPutRequest struct {
	Version     int                       `json:"version"`
	Partitions  []SessionSidebarPartition `json:"partitions"`
	Assignments map[string]string         `json:"assignments"`
	TraceID     string                    `json:"trace_id,omitempty"`
}

type SessionIDParams struct {
	ID string `json:"id"`
}

type SessionGetParams struct {
	ID     string `json:"id"`
	Limit  *int   `json:"limit,omitempty"`
	Before *int   `json:"before,omitempty"`
}

type SessionDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}
