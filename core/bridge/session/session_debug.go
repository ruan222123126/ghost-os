package session

import (
	"log"
	"os"
	"strings"
)

var debugNilSessionReceiver = envBool("GHOST_BRIDGE_DEBUG") || envBool("GHOST_DEBUG") || envBool("DEBUG")

func debugNilReceiver(method string) {
	if !debugNilSessionReceiver {
		return
	}
	log.Printf("[SESSION] nil Session receiver (caller bug?): method=%s", strings.TrimSpace(method))
}

func envBool(key string) bool {
	value := strings.TrimSpace(os.Getenv(strings.TrimSpace(key)))
	if value == "" {
		return false
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
