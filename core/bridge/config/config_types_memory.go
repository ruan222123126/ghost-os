package config

type MemoryAugmentationConfig struct {
	Enabled             bool
	LearningEnabled     bool
	RecallEnabled       bool
	MaxRecallItems      int
	MinConfidence       float64
	SessionScopeEnabled bool
	UserScopeEnabled    bool
	LLMModel            string
	UserScopeID         string
}
