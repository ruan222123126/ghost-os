package config

type Store interface {
	Config() (Config, error)
	Snapshot() Snapshot
	PublicSnapshot() (Snapshot, error)
	ListProviders() ([]ProviderRecord, error)
	ListProviderSyncRecords() ([]ProviderRecord, error)
	GetProvider(name string) (ProviderRecord, error)
	GetProviderByID(providerID string) (ProviderRecord, error)
	AddProvider(ProviderRecord) error
	UpdateProvider(name string, cfg ProviderRecord) error
	DeleteProvider(name string) error
	SetActiveProvider(name string) error
	ListTools() ([]ToolRecord, error)
	UpdateTool(ToolUpdateRequest) error
	SetSkillEnabled(skillID string, enabled bool) error
	SystemPrompts() (SystemPromptFiles, error)
	UpdateSystemPrompts(SystemPromptUpdateRequest) (SystemPromptFiles, error)
	Presets() ([]Preset, error)
	CreatePreset(PresetCreateRequest) (Preset, error)
	UpdatePreset(string, PresetUpdateRequest) (Preset, error)
	DeletePreset(string) (Preset, error)
	ApplyPreset(string) (Preset, error)
	Update(UpdateRequest) error
	SetProjectRoot(path string) error
	WithRuntimeOverrides(providerName string, model string) (Store, error)
}

func NewStoreFromEnv() (Store, error) {
	return newStoreFromEnv()
}
