// Request/response codec utilities for validating and normalizing bus envelopes.

package orchestration

import (
	"encoding/json"

	appservice "ghost-os/bridge/orchestration/internal/app/service"
)

// decodeActionParams 将 raw params 解码为用例参数类型。
func decodeActionParams[T any](raw json.RawMessage) (T, error) {
	return appservice.DecodeActionParams[T](raw)
}

// validateBusRequest 做 envelope 级别校验，要求 action/trace_id 存在且 params 为对象。
func validateBusRequest(req apiRequest) error {
	return appservice.ValidateBusRequest(req)
}

// decodeParams 解码 params 并禁止未知字段；空/null 统一按空对象处理。
func decodeParams(raw json.RawMessage, target any) error {
	return appservice.DecodeParams(raw, target)
}
