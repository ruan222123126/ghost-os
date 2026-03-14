package execution

import "strings"

const (
	nativeAllowedReadEnv  = "GHOST_NATIVE_ALLOWED_READ_PATHS"
	nativeAllowedWriteEnv = "GHOST_NATIVE_ALLOWED_WRITE_PATHS"
)

func buildNativeAllowedPathEnv(readPaths []string, writePaths []string) []string {
	read := joinPathList(readPaths)
	write := joinPathList(writePaths)
	if read == "" && write == "" {
		return nil
	}
	env := make([]string, 0, 2)
	if read != "" {
		env = append(env, nativeAllowedReadEnv+"="+read)
	}
	if write != "" {
		env = append(env, nativeAllowedWriteEnv+"="+write)
	}
	return env
}

func joinPathList(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, ",")
}
