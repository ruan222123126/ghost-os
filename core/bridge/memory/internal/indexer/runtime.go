package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Runtime struct {
	config      Config
	source      BucketSource
	checkpoints CheckpointStore
	projectors  []Projector
	scheduler   BucketScheduler

	startOnce sync.Once
	stopOnce  sync.Once
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

func NewRuntime(config Config, source BucketSource, checkpoints CheckpointStore, projectors ...Projector) *Runtime {
	active := make([]Projector, 0, len(projectors))
	for _, projector := range projectors {
		if projector == nil || !projector.Enabled() {
			continue
		}
		active = append(active, projector)
	}
	return &Runtime{
		config:      normalizeConfig(config),
		source:      source,
		checkpoints: checkpoints,
		projectors:  active,
		scheduler:   BucketScheduler{},
		stopCh:      make(chan struct{}),
	}
}

func (r *Runtime) Enabled() bool {
	return r != nil && r.config.Enabled && r.source != nil && r.checkpoints != nil && len(r.projectors) > 0
}

func (r *Runtime) Start() {
	if !r.Enabled() {
		return
	}
	r.startOnce.Do(func() {
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			ticker := time.NewTicker(r.config.PollInterval)
			defer ticker.Stop()
			for {
				select {
				case <-r.stopCh:
					return
				case <-ticker.C:
					_, _ = r.runOnce(context.Background())
				}
			}
		}()
	})
}

func (r *Runtime) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
	r.wg.Wait()
}

func (r *Runtime) RunOnce(ctx context.Context) error {
	_, err := r.runOnce(ctx)
	return err
}

func (r *Runtime) runOnce(ctx context.Context) (int, error) {
	if !r.Enabled() {
		return 0, nil
	}
	buckets, err := r.source.ListBuckets(ctx)
	if err != nil {
		return 0, err
	}
	ordered := r.scheduler.Order(buckets)
	if len(ordered) == 0 {
		return 0, nil
	}
	workers := r.config.Workers
	if workers <= 0 {
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	processed := 0
	var firstErr error
	for _, bucket := range ordered {
		bucket := bucket
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			count, err := r.processBucket(ctx, bucket)
			mu.Lock()
			defer mu.Unlock()
			processed += count
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}()
	}
	wg.Wait()
	return processed, firstErr
}

func (r *Runtime) processBucket(ctx context.Context, bucket Bucket) (int, error) {
	events, err := r.source.LoadBucket(ctx, bucket)
	if err != nil {
		return 0, err
	}
	envelopes := BuildTurnEnvelopes(bucket, events)
	if len(envelopes) == 0 {
		return 0, nil
	}
	processed := 0
	for _, projector := range r.projectors {
		checkpoint, err := r.checkpoints.Load(projector.Name(), bucket)
		if err != nil {
			return processed, err
		}
		checkpoint = normalizeCheckpoint(checkpoint)
		if checkpoint.Segment != "" && checkpoint.Segment != normalizeBucket(bucket).SegmentPath {
			checkpoint = Checkpoint{}
		}
		batched := 0
		for _, envelope := range envelopes {
			if envelope.CommitOffset <= checkpoint.Offset {
				continue
			}
			if err := projector.Process(ctx, envelope); err != nil {
				checkpoint.ErrorCount++
				_ = r.checkpoints.Save(projector.Name(), bucket, checkpoint)
				return processed, fmt.Errorf("projector %s: %w", projector.Name(), err)
			}
			checkpoint = Checkpoint{
				Segment:     normalizeBucket(bucket).SegmentPath,
				Offset:      envelope.CommitOffset,
				LastEventID: firstNonEmpty(envelope.LastEventID, envelope.CommitEventID),
				ErrorCount:  checkpoint.ErrorCount,
			}
			if err := r.checkpoints.Save(projector.Name(), bucket, checkpoint); err != nil {
				return processed, err
			}
			processed++
			batched++
			if batched >= r.config.BatchSize {
				break
			}
		}
		if builder, ok := projector.(ViewManifestBuilder); ok {
			if store, ok := r.checkpoints.(ViewManifestStore); ok {
				manifest, err := builder.BuildViewManifest(ctx, bucket, envelopes)
				if err != nil {
					return processed, err
				}
				if err := store.SaveViewManifest(projector.Name(), bucket, manifest); err != nil {
					return processed, err
				}
			}
		}
	}
	return processed, nil
}
