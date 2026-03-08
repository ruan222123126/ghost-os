package memory

import "time"

const (
	defaultAutoRecallLimit   = 5
	defaultEvolutionInterval = time.Hour
	defaultRecipeMinSuccess  = 0.66
	defaultIndexPollInterval = 30 * time.Second
)

type WarmConfig struct {
	Capacity              int
	Path                  string
	TTL                   time.Duration
	AutoRecallEnabled     bool
	AutoRecallLimit       int
	TemporalDecayEnabled  bool
	TemporalDecayHalfLife time.Duration
	AnchorEnabled         bool
	AnchorMinWeight       float64
	EvolutionInterval     time.Duration
	EvolutionEnabled      bool
	EvolutionUseWorker    bool
	EvolutionBatchSize    int
}

type ColdConfig struct {
	BaseDir string
}

type LedgerConfig struct {
	BaseDir       string
	Namespace     string
	WorkspaceID   string
	DualWrite     bool
	ReadEnabled   bool
	ShadowCompare bool
}

type IndexConfig struct {
	Enabled         bool
	Workers         int
	BatchSize       int
	PollInterval    time.Duration
	CheckpointDir   string
	ShadowCompare   bool
	LegacyColdAsync bool
}

type RecallConfig struct {
	IntentPlannerEnabled      bool
	ShadowEnabled             bool
	HybridRerankEnabled       bool
	RerankDebugEnabled        bool
	RecallInjectMinConfidence float64
	ConflictPenalty           float64
	BucketReadEnabled         bool
	BucketShadowCompare       bool
	MaxBuckets                int
	MaxRecentBuckets          int
	MaxHistoricalBuckets      int
	BucketSessionBudget       int
	BucketDebug               bool
	CompressionEnabled        bool
	CompressionMinSupport     int
	CompressionAgeTiers       []CompressionAgeTier
}

type CompressionAgeTier struct {
	Name            string        `json:"name,omitempty"`
	MinAge          time.Duration `json:"min_age,omitempty"`
	MaxAge          time.Duration `json:"max_age,omitempty"`
	PreferredLayers []string      `json:"preferred_layers,omitempty"`
}

type TruthConfig struct {
	Enabled                       bool
	DualWrite                     bool
	BaseDir                       string
	ShadowFailOpen                bool
	ReadEnabled                   bool
	TopK                          int
	MinSupportRefs                int
	SchemaVersion                 int
	ClaimArbitrationEnabled       bool
	ClaimStatusProjectionEnabled  bool
	LegacyObjectProjectionEnabled bool
}

type GraphConfig struct {
	Enabled          bool
	Path             string
	ExtractOnArchive bool
	ExtractOnEvolve  bool
	MaxHops          int
	MaxHits          int
	MinConfidence    float64
	Namespace        string
	DebugEnabled     bool
}

type DecisionRecipeConfig struct {
	Enabled                     bool
	Interval                    time.Duration
	MinSupport                  int
	ReuseEnabled                bool
	ReuseEnabledSet             bool
	ExecutionTrackingEnabled    bool
	ExecutionTrackingEnabledSet bool
	BackfillEnabled             bool
	BackfillEnabledSet          bool
	DefaultEnabled              bool
	DefaultEnabledSet           bool
	DefaultGrayPercent          int
	MinSelectionConfidence      float64
	MinSuccessRate              float64
	BackfillBatchSize           int
	BackfillInterval            time.Duration
}

type DecisionConfig struct {
	Enabled          bool
	CaptureOnTurn    bool
	CaptureOnTurnSet bool
	Path             string
	MaxHits          int
	MinConfidence    float64
	MinReuseScore    float64
	DebugEnabled     bool
	Recipe           DecisionRecipeConfig
}

type VectorConfig struct {
	Enabled  bool
	Path     string
	TopK     int
	MinScore float64
}

type HygieneConfig struct {
	Enabled bool
	Path    string
}

type RuntimeConfig struct {
	SessionStore SessionStorePort
	Summarizer   Summarizer
}

// MemoryConfig 定义三层记忆管理器初始化参数，并按能力拆成小配置块。
type MemoryConfig struct {
	Warm     WarmConfig
	Cold     ColdConfig
	Ledger   LedgerConfig
	Index    IndexConfig
	Recall   RecallConfig
	Truth    TruthConfig
	Graph    GraphConfig
	Decision DecisionConfig
	Vector   VectorConfig
	Hygiene  HygieneConfig
	Runtime  RuntimeConfig

	WarmCapacity                       int
	WarmPath                           string
	ColdBaseDir                        string
	LedgerBaseDir                      string
	LedgerNamespace                    string
	LedgerWorkspaceID                  string
	LedgerDualWrite                    bool
	LedgerReadEnabled                  bool
	LedgerShadowCompare                bool
	TruthEnabled                       bool
	TruthDualWrite                     bool
	TruthBaseDir                       string
	TruthShadowFailOpen                bool
	TruthReadEnabled                   bool
	TruthTopK                          int
	TruthMinSupportRefs                int
	TruthSchemaVersion                 int
	TruthClaimArbitrationEnabled       bool
	TruthClaimStatusProjectionEnabled  bool
	TruthLegacyObjectProjectionEnabled bool
	IntentPlannerEnabled               bool
	VectorEnabled                      bool
	VectorPath                         string
	VectorTopK                         int
	VectorMinScore                     float64
	HygieneEnabled                     bool
	HygienePath                        string
	ShadowRecallEnabled                bool
	HybridRerankEnabled                bool
	RerankDebugEnabled                 bool
	RecallInjectMinConfidence          float64
	ConflictPenalty                    float64
	BucketReadEnabled                  bool
	BucketShadowCompare                bool
	MaxBuckets                         int
	MaxRecentBuckets                   int
	MaxHistoricalBuckets               int
	BucketSessionBudget                int
	BucketDebug                        bool
	CompressionEnabled                 bool
	CompressionMinSupport              int
	CompressionAgeTiers                []CompressionAgeTier
	AutoRecallEnabled                  bool
	AutoRecallLimit                    int
	WarmTTL                            time.Duration
	TemporalDecayEnabled               bool
	TemporalDecayHalfLife              time.Duration
	AnchorEnabled                      bool
	AnchorMinWeight                    float64
	EvolutionInterval                  time.Duration
	EvolutionEnabled                   bool
	EvolutionUseWorker                 bool
	EvolutionBatchSize                 int
	GraphEnabled                       bool
	GraphPath                          string
	GraphExtractOnArchive              bool
	GraphExtractOnEvolve               bool
	GraphMaxHops                       int
	GraphMaxHits                       int
	GraphMinConfidence                 float64
	GraphNamespace                     string
	GraphDebugEnabled                  bool
	DecisionEnabled                    bool
	DecisionCaptureOnTurn              bool
	DecisionCaptureOnTurnSet           bool
	DecisionPath                       string
	DecisionMaxHits                    int
	DecisionMinConfidence              float64
	DecisionMinReuseScore              float64
	DecisionRecipeEnabled              bool
	DecisionRecipeInterval             time.Duration
	DecisionRecipeMinSupport           int
	DecisionDebugEnabled               bool
	RecipeReuseEnabled                 bool
	RecipeReuseEnabledSet              bool
	RecipeExecutionTrackingEnabled     bool
	RecipeExecutionTrackingEnabledSet  bool
	RecipeBackfillEnabled              bool
	RecipeBackfillEnabledSet           bool
	RecipeDefaultEnabled               bool
	RecipeDefaultEnabledSet            bool
	RecipeDefaultGrayPercent           int
	RecipeMinSelectionConfidence       float64
	RecipeMinSuccessRate               float64
	RecipeBackfillBatchSize            int
	RecipeBackfillInterval             time.Duration
	SessionStore                       SessionStorePort
	Summarizer                         Summarizer
}
