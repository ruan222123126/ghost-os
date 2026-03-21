package orchestration

import bridgeconfig "ghost-os/bridge/config"

// WrapConfigStore 将 config 包中的运行态存储适配为 orchestration 使用的包装类型。
func WrapConfigStore(store *bridgeconfig.Store) *ConfigStore {
	if store == nil {
		return nil
	}
	return &ConfigStore{inner: store}
}
