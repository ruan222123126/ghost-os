package orchestration

type serviceLifecycle struct {
	runtimes    *serviceRuntimeState
	sessionPush *sessionPushHub
}

func newServiceLifecycle(runtimes *serviceRuntimeState, sessionPush *sessionPushHub) *serviceLifecycle {
	return &serviceLifecycle{
		runtimes:    runtimes,
		sessionPush: sessionPush,
	}
}

func (l *serviceLifecycle) close() {
	if l == nil {
		return
	}
	if l.runtimes != nil {
		l.runtimes.close()
	}
	if l.sessionPush != nil {
		l.sessionPush.Close()
	}
}

func (l *serviceLifecycle) sessionPushHub() *sessionPushHub {
	if l == nil {
		return nil
	}
	return l.sessionPush
}
