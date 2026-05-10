package orchestration

import "ghost-os/bridge/orchestration/internal/contracts/api"

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams = api.AgentParams
type agentStopParams = api.AgentStopParams
type sessionIDParams = api.SessionIDParams
type sessionGetParams = api.SessionGetParams
type sessionDeleteResponse = api.SessionDeleteResponse

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput
