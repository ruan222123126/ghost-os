package memory

import "context"

func (r *IndexRuntime) Enabled() bool {
	return r != nil && r.inner != nil && r.inner.Enabled()
}

func (r *IndexRuntime) Start() {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.Start()
}

func (r *IndexRuntime) Stop() {
	if r == nil || r.inner == nil {
		return
	}
	r.inner.Stop()
}

func (r *IndexRuntime) RunOnce(ctx context.Context) error {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.RunOnce(ctx)
}

func (r *IndexRuntime) OwnsProjector(name string) bool {
	if r == nil || len(r.owns) == 0 {
		return false
	}
	return r.owns[name]
}

func (m *MemoryManager) RunIndexOnce(ctx context.Context) error {
	if m == nil || m.indexer == nil {
		return nil
	}
	return m.indexer.RunOnce(ctx)
}
