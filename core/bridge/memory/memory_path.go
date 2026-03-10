// Memory path helpers centralize on-disk layout for warm/cold memory persistence.

package memory

import "ghost-os/bridge/memory/internal/pathutil"

func resolveMemoryPath(pathValue string) string {
	return pathutil.Resolve(pathValue)
}

func resolveMemoryPathWithError(pathValue string) (string, error) {
	return pathutil.ResolveWithError(pathValue)
}
