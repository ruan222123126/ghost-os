package memory

import (
	"reflect"

	"ghost-os/bridge/session"
)

// SessionStorePort 只暴露记忆归档阶段需要的最小会话读取能力。
type SessionStorePort interface {
	Load(sessionID string) (*session.Session, error)
}

func hasSessionStore(store SessionStorePort) bool {
	if store == nil {
		return false
	}
	value := reflect.ValueOf(store)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return !value.IsNil()
	default:
		return true
	}
}
