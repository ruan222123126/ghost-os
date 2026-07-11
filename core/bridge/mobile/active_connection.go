package mobile

import (
	"strings"
	"sync"
	"time"
)

type ActiveConnection struct {
	DeviceID     string
	ConnectionID string
	StartedAt    time.Time
	Cancel       func()
}

type AcquireResult struct {
	Accepted bool
	Busy     bool
	Takeover bool
	Active   ActiveConnection
}

type ActiveConnectionGate struct {
	mu     sync.Mutex
	active *ActiveConnection
}

func NewActiveConnectionGate() *ActiveConnectionGate {
	return &ActiveConnectionGate{}
}

func (g *ActiveConnectionGate) Acquire(next ActiveConnection) AcquireResult {
	g.mu.Lock()
	defer g.mu.Unlock()

	next.DeviceID = strings.TrimSpace(next.DeviceID)
	next.ConnectionID = strings.TrimSpace(next.ConnectionID)
	if next.StartedAt.IsZero() {
		next.StartedAt = time.Now()
	}
	if g.active == nil {
		g.active = cloneActiveConnection(next)
		return AcquireResult{Accepted: true, Active: *g.active}
	}
	if g.active.DeviceID != next.DeviceID {
		return AcquireResult{Busy: true, Active: *g.active}
	}
	previous := *g.active
	g.active = cloneActiveConnection(next)
	if previous.Cancel != nil {
		previous.Cancel()
	}
	return AcquireResult{Accepted: true, Takeover: true, Active: *g.active}
}

func (g *ActiveConnectionGate) Release(deviceID string, connectionID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active == nil {
		return
	}
	if g.active.DeviceID != strings.TrimSpace(deviceID) {
		return
	}
	if g.active.ConnectionID != strings.TrimSpace(connectionID) {
		return
	}
	g.active = nil
}

func (g *ActiveConnectionGate) Snapshot() (ActiveConnection, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active == nil {
		return ActiveConnection{}, false
	}
	return *g.active, true
}

func cloneActiveConnection(raw ActiveConnection) *ActiveConnection {
	value := raw
	return &value
}
