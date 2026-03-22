package config

type Store interface {
	Config() (Config, error)
	Snapshot() Snapshot
	ListProviders() []ProviderRecord
	AddProvider(ProviderRecord) error
	UpdateProvider(name string, cfg ProviderRecord) error
	DeleteProvider(name string) error
	SetActiveProvider(name string) error
	Update(UpdateRequest) error
	SetProjectRoot(path string) error
}

func NewStoreFromEnv() (Store, error) {
	return newStoreFromEnv()
}
