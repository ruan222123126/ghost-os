package orchestration

import sharedtext "ghost-os/bridge/orchestration/internal/shared/text"

func truncateRunes(value string, limit int) string {
	return sharedtext.TruncateRunes(value, limit)
}
