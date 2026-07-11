package tasks

const (
	LoadIssueInvalidFilename = "invalid_filename"
	LoadIssueReadError       = "read_error"
	LoadIssueDecodeError     = "decode_error"
	LoadIssueInvalidConfig   = "invalid_config"
	LoadIssueIDMismatch      = "id_mismatch"
)

type LoadIssue struct {
	Kind   string `json:"kind,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error"`
}
