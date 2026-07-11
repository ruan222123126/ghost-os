package trace

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestRunRegistryRegisterSuccess(t *testing.T) {
	registry := NewRunRegistry()
	cancelled := make(chan struct{}, 1)

	err := registry.Register("session-1", "trace-1", func() {
		cancelled <- struct{}{}
	})
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if !registry.IsInflight("session-1") {
		t.Fatal("session should be inflight")
	}
	if registry.Count() != 1 {
		t.Fatalf("unexpected count: got %d want %d", registry.Count(), 1)
	}
	handle := registry.GetBySessionID("session-1")
	if handle == nil {
		t.Fatal("expected run handle")
	}
	if handle.TraceID != "trace-1" {
		t.Fatalf("unexpected trace id: got %q want %q", handle.TraceID, "trace-1")
	}
	select {
	case <-cancelled:
		t.Fatal("cancel should not be called on register")
	default:
	}
}

func TestRunRegistryRegisterDuplicate(t *testing.T) {
	registry := NewRunRegistry()
	if err := registry.Register("session-1", "trace-1", func() {}); err != nil {
		t.Fatalf("first Register returned error: %v", err)
	}

	err := registry.Register("session-1", "trace-2", func() {})
	if !errors.Is(err, ErrSessionInflight) {
		t.Fatalf("unexpected error: got %v want %v", err, ErrSessionInflight)
	}
}

func TestRunRegistryRegisterNilReceiverReturnsError(t *testing.T) {
	var registry *RunRegistry
	if err := registry.Register("session-1", "trace-1", func() {}); !errors.Is(err, ErrRunRegistryNil) {
		t.Fatalf("unexpected error: got %v want %v", err, ErrRunRegistryNil)
	}
}

func TestRunRegistryUnregister(t *testing.T) {
	registry := NewRunRegistry()
	if err := registry.Register("session-1", "trace-1", func() {}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	registry.Unregister("session-1")
	if registry.IsInflight("session-1") {
		t.Fatal("session should not be inflight after unregister")
	}
	if registry.Count() != 0 {
		t.Fatalf("unexpected count: got %d want %d", registry.Count(), 0)
	}
	if err := registry.Register("session-1", "trace-2", func() {}); err != nil {
		t.Fatalf("Register after unregister returned error: %v", err)
	}
}

func TestRunRegistryCancelBySessionID(t *testing.T) {
	registry := NewRunRegistry()
	cancelled := make(chan struct{}, 1)
	if err := registry.Register("session-1", "trace-1", func() {
		cancelled <- struct{}{}
	}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if err := registry.CancelBySessionID("session-1"); err != nil {
		t.Fatalf("CancelBySessionID returned error: %v", err)
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("cancel func was not called")
	}
	if registry.Count() != 1 {
		t.Fatalf("unexpected count before unregister: got %d want %d", registry.Count(), 1)
	}
	registry.Unregister("session-1")
	if registry.Count() != 0 {
		t.Fatalf("unexpected count after unregister: got %d want %d", registry.Count(), 0)
	}
}

func TestRunRegistryCancelByTraceID(t *testing.T) {
	registry := NewRunRegistry()
	cancelled := make(chan struct{}, 1)
	if err := registry.Register("session-1", "trace-1", func() {
		cancelled <- struct{}{}
	}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	if err := registry.CancelByTraceID("trace-1"); err != nil {
		t.Fatalf("CancelByTraceID returned error: %v", err)
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("cancel func was not called")
	}
	if registry.GetByTraceID("trace-1") == nil {
		t.Fatal("trace handle should remain until unregister")
	}
	registry.Unregister("session-1")
	if registry.GetByTraceID("trace-1") != nil {
		t.Fatal("trace handle should be removed after unregister")
	}
}

func TestRunRegistryCancelAndWaitByTraceID(t *testing.T) {
	registry := NewRunRegistry()
	cancelled := make(chan struct{}, 1)
	if err := registry.Register("session-1", "trace-1", func() {
		cancelled <- struct{}{}
	}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	go func() {
		<-cancelled
		registry.Unregister("session-1")
	}()

	handle, err := registry.CancelAndWaitByTraceID(context.Background(), "trace-1")
	if err != nil {
		t.Fatalf("CancelAndWaitByTraceID returned error: %v", err)
	}
	if handle == nil {
		t.Fatal("expected run handle")
	}
	if handle.SessionID != "session-1" || handle.TraceID != "trace-1" {
		t.Fatalf("unexpected handle: %+v", handle)
	}
	if registry.Count() != 0 {
		t.Fatalf("unexpected count after wait: got %d want %d", registry.Count(), 0)
	}
}

func TestRunRegistryCancelNotFound(t *testing.T) {
	registry := NewRunRegistry()
	if err := registry.CancelBySessionID("missing"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("unexpected session cancel error: got %v want %v", err, ErrRunNotFound)
	}
	if err := registry.CancelByTraceID("missing"); !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("unexpected trace cancel error: got %v want %v", err, ErrRunNotFound)
	}
}

func TestRunRegistryConcurrent(t *testing.T) {
	registry := NewRunRegistry()
	var active atomic.Int32
	var wg sync.WaitGroup

	for index := 0; index < 32; index++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessionID := "session-shared"
			traceID := "trace-" + string(rune('a'+i))

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := registry.Register(sessionID, traceID, func() {
				active.Add(-1)
				cancel()
			})
			if err != nil {
				if !errors.Is(err, ErrSessionInflight) {
					t.Errorf("unexpected register error: %v", err)
				}
				return
			}

			active.Add(1)
			if got := registry.Count(); got < 1 {
				t.Errorf("unexpected count after register: %d", got)
			}
			if err := registry.CancelBySessionID(sessionID); err != nil && !errors.Is(err, ErrRunNotFound) {
				t.Errorf("unexpected cancel error: %v", err)
			}
			<-ctx.Done()
			registry.Unregister(sessionID)
		}(index)
	}

	wg.Wait()
	if got := active.Load(); got != 0 {
		t.Fatalf("unexpected active count: got %d want %d", got, 0)
	}
	if got := registry.Count(); got != 0 {
		t.Fatalf("unexpected final count: got %d want %d", got, 0)
	}
}
