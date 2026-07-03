package runtimeutil

import (
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/internal/stringutil"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func BuildGhostRuntimeSelection(
	store bridgeconfig.Store,
	cfg bridgeconfig.Config,
	mode string,
) *session.RuntimeSelection {
	return BuildGhostRuntimeSelectionFromSnapshot(storeSnapshot(store), cfg, mode)
}

func BuildGhostRuntimeSelectionFromSnapshot(
	snapshot bridgeconfig.Snapshot,
	cfg bridgeconfig.Config,
	mode string,
) *session.RuntimeSelection {
	providerType := stringutil.FirstNonEmpty(snapshot.ProviderType, string(cfg.Provider.Type))
	if providerType == "" {
		providerType = string(llm.ProviderCustom)
	}
	selection, ok := session.NormalizeRuntimeSelection(session.RuntimeSelection{
		Runtime:      session.RuntimeSelectionGhost,
		Provider:     snapshot.Provider,
		ProviderType: providerType,
		Model:        stringutil.FirstNonEmpty(snapshot.Model, cfg.Provider.Model),
		Mode:         mode,
	})
	if !ok {
		return nil
	}
	return &selection
}

func storeSnapshot(store bridgeconfig.Store) bridgeconfig.Snapshot {
	if store == nil {
		return bridgeconfig.Snapshot{}
	}
	return store.Snapshot()
}
