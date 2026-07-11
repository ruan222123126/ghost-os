package sessions

import (
	"fmt"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
)

func validateSessionGetPageParams(params api.SessionGetParams) error {
	invalidLimit := params.Limit != nil && (*params.Limit <= 0 || *params.Limit > session.MaxDetailPageLimit)
	if invalidLimit {
		return fmt.Errorf(
			"%w: limit must be an integer between 1 and %d",
			ErrInvalidSessionPageQuery,
			session.MaxDetailPageLimit,
		)
	}
	if params.Before != nil && *params.Before < 0 {
		return fmt.Errorf("%w: before must be a non-negative integer", ErrInvalidSessionPageQuery)
	}
	return nil
}
