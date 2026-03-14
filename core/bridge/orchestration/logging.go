package orchestration

import "log"

func logAction(traceID string, action string, status string, err error) {
	if err != nil {
		log.Printf("trace_id=%s action=%s status=%s error=%v", traceID, action, status, err)
		return
	}
	log.Printf("trace_id=%s action=%s status=%s", traceID, action, status)
}
