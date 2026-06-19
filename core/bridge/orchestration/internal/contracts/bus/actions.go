package bus

const (
	ActionAgentSend          = "AGENT_SEND"
	ActionAgentStop          = "AGENT_STOP"
	ActionHumanResponse      = "HUMAN_RESPONSE"
	ActionSessionsList       = "SESSIONS_LIST"
	ActionSessionGet         = "SESSION_GET"
	ActionConfigGet          = "CONFIG_GET"
	ActionConfigUpdate       = "CONFIG_UPDATE"
	ActionConfigProvidersGet = "CONFIG_PROVIDERS_GET"
	ActionTaskCreate         = "TASK_CREATE"
	ActionTaskList           = "TASK_LIST"
	ActionTaskGet            = "TASK_GET"
	ActionTaskUpdate         = "TASK_UPDATE"
	ActionTaskRunNow         = "TASK_RUN_NOW"
	ActionTaskStop           = "TASK_STOP"
	ActionTaskLogs           = "TASK_LOGS"
	ActionTaskDelete         = "TASK_DELETE"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"
)

const AssistantSessionEndSignal = "END_SESSION"
