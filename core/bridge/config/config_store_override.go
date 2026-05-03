package config

import "strings"

type projectRootOverrideStore struct {
	Store
	projectRoot string
}

func WithProjectRootOverride(store Store, projectRoot string) Store {
	if store == nil {
		return nil
	}
	trimmed := strings.TrimSpace(projectRoot)
	if trimmed == "" {
		return store
	}
	return projectRootOverrideStore{
		Store:       store,
		projectRoot: trimmed,
	}
}

func (s projectRootOverrideStore) Config() (Config, error) {
	cfg, err := s.Store.Config()
	if err != nil {
		return Config{}, err
	}
	cfg.ProjectRoot = s.projectRoot
	return cfg, nil
}
