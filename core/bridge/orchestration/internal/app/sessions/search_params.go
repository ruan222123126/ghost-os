package sessions

import (
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
)

func ResolveSessionSearchParams(params api.SessionSearchParams) (string, int, error) {
	if params.Limit != nil && *params.Limit <= 0 {
		return "", 0, fmt.Errorf("%w: limit must be a positive integer", ErrInvalidSessionSearchQuery)
	}

	limit := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	return strings.TrimSpace(params.Query), limit, nil
}
