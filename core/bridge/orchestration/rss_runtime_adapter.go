package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
)

func newRSSInboxServiceFromConfig(store bridgeconfig.Store) (*bridgerss.RSSInboxService, error) {
	service, err := bridgerss.NewRSSInboxServiceFromConfig(store)
	if err != nil {
		return nil, err
	}
	service.SetReportBuilder(newRuntimeRSSReportBuilder(store))
	return service, nil
}
