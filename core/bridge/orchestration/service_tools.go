package orchestration

import apptools "ghost-os/bridge/orchestration/internal/app/tools"

const (
	busActionToolList   = apptools.ActionList
	busActionToolUpdate = apptools.ActionUpdate
)

func (s *bridgeService) executeToolListAction(traceID string) (any, int, error) {
	return s.toolService().List(traceID)
}

func (s *bridgeService) executeToolUpdateAction(
	params toolNameParams,
	req toolUpdateRequest,
	traceID string,
) (any, int, error) {
	return s.toolService().Update(params, req, traceID)
}

func (s *bridgeService) toolService() apptools.Service {
	return apptools.Service{
		Store:          s.configStore,
		SchemaProvider: toolSchemaProvider{service: s},
		Logger:         serviceActionLogger{},
	}
}

type toolSchemaProvider struct {
	service *bridgeService
}

func (p toolSchemaProvider) ListToolInputSchemas() (map[string]map[string]any, error) {
	return p.service.listToolInputSchemas()
}

func mapToolConfigError(err error) int {
	return apptools.MapConfigError(err)
}

func cloneOptionalInt(raw *int) *int {
	return apptools.CloneOptionalInt(raw)
}
