package bus

const (
	ActionAgentSend            = "AGENT_SEND"
	ActionAgentStop            = "AGENT_STOP"
	ActionExternalAgentStart   = "EXTERNAL_AGENT_START"
	ActionExternalAgentSend    = "EXTERNAL_AGENT_SEND"
	ActionExternalAgentStop    = "EXTERNAL_AGENT_STOP"
	ActionExternalAgentApprove = "EXTERNAL_AGENT_APPROVE"
	ActionHumanResponse        = "HUMAN_RESPONSE"
	ActionSessionsList         = "SESSIONS_LIST"
	ActionSessionsSearch       = "SESSIONS_SEARCH"
	ActionSessionGet           = "SESSION_GET"
	ActionSessionAppend        = "SESSION_APPEND"
	ActionConfigGet            = "CONFIG_GET"
	ActionConfigUpdate         = "CONFIG_UPDATE"
	ActionConfigProvidersGet   = "CONFIG_PROVIDERS_GET"
	ActionConfigProviderExport = "CONFIG_PROVIDER_EXPORT"
	ActionConfigProviderCreate = "CONFIG_PROVIDER_CREATE"
	ActionConfigProviderUpdate = "CONFIG_PROVIDER_UPDATE"
	ActionConfigProviderDelete = "CONFIG_PROVIDER_DELETE"
	ActionSkillList            = "SKILL_LIST"
	ActionSkillUpdate          = "SKILL_UPDATE"
	ActionSkillDelete          = "SKILL_DELETE"
	ActionTaskCreate           = "TASK_CREATE"
	ActionTaskList             = "TASK_LIST"
	ActionTaskGet              = "TASK_GET"
	ActionTaskUpdate           = "TASK_UPDATE"
	ActionTaskRunNow           = "TASK_RUN_NOW"
	ActionTaskStop             = "TASK_STOP"
	ActionTaskLogs             = "TASK_LOGS"
	ActionTaskDelete           = "TASK_DELETE"
)

const (
	StatusSuccess = "success"
	StatusError   = "error"
)

const AssistantSessionEndSignal = "END_SESSION"
