package indexer

import (
	"context"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const checkpointFileName = "checkpoint.json"

type Config struct {
	Enabled         bool
	Workers         int
	BatchSize       int
	PollInterval    time.Duration
	ShadowCompare   bool
	LegacyColdAsync bool
}

type Bucket struct {
	Namespace   string
	Workspace   string
	Month       string
	RootDir     string
	SegmentPath string
}

type EventPayload struct {
	Message      *llm.Message
	StartIndex   int
	MessageCount int
}

type Event struct {
	EventID      string
	TraceID      string
	Namespace    string
	Workspace    string
	SessionID    string
	TurnID       string
	Kind         string
	MessageIndex int
	OccurredAt   time.Time
	Offset       int64
	Payload      EventPayload
}

type TurnMessage struct {
	Index   int
	EventID string
	Offset  int64
	Message llm.Message
}

type TurnEnvelope struct {
	Bucket         Bucket
	Namespace      string
	Workspace      string
	SessionID      string
	TurnID         string
	TraceID        string
	StartIndex     int
	MessageCount   int
	Messages       []llm.Message
	MessageEvents  []TurnMessage
	CommitEventID  string
	CommitOffset   int64
	LastEventID    string
	StartedAt      time.Time
	CommittedAt    time.Time
	CommittedIndex int
}

type Checkpoint struct {
	Segment     string    `json:"segment,omitempty"`
	Offset      int64     `json:"offset,omitempty"`
	LastEventID string    `json:"last_event_id,omitempty"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	ErrorCount  int       `json:"error_count,omitempty"`
}

type BucketSource interface {
	ListBuckets(ctx context.Context) ([]Bucket, error)
	LoadBucket(ctx context.Context, bucket Bucket) ([]Event, error)
}

type CheckpointStore interface {
	Load(projector string, bucket Bucket) (Checkpoint, error)
	Save(projector string, bucket Bucket, checkpoint Checkpoint) error
}

type Projector interface {
	Name() string
	Enabled() bool
	Process(ctx context.Context, envelope TurnEnvelope) error
}

type ViewManifestBuilder interface {
	BuildViewManifest(ctx context.Context, bucket Bucket, envelopes []TurnEnvelope) (ViewManifest, error)
}

func normalizeConfig(cfg Config) Config {
	if cfg.Workers <= 0 {
		cfg.Workers = 1
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 30 * time.Second
	}
	return cfg
}

func normalizeBucket(bucket Bucket) Bucket {
	bucket.Namespace = strings.TrimSpace(bucket.Namespace)
	bucket.Workspace = strings.TrimSpace(bucket.Workspace)
	bucket.Month = strings.TrimSpace(bucket.Month)
	bucket.RootDir = strings.TrimSpace(bucket.RootDir)
	bucket.SegmentPath = strings.TrimSpace(bucket.SegmentPath)
	return bucket
}

func normalizeCheckpoint(checkpoint Checkpoint) Checkpoint {
	checkpoint.Segment = strings.TrimSpace(checkpoint.Segment)
	checkpoint.LastEventID = strings.TrimSpace(checkpoint.LastEventID)
	checkpoint.UpdatedAt = checkpoint.UpdatedAt.UTC()
	if checkpoint.ErrorCount < 0 {
		checkpoint.ErrorCount = 0
	}
	return checkpoint
}
