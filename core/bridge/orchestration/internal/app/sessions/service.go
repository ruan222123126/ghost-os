package sessions

import (
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
)

const (
	ActionList                 = "SESSIONS_LIST"
	ActionSearch               = "SESSIONS_SEARCH"
	ActionGet                  = "SESSION_GET"
	ActionDelete               = "SESSION_DELETE"
	ActionSources              = "SESSION_SOURCES"
	ActionSidebarPartitionsGet = "SESSION_PARTITIONS_GET"
	ActionSidebarPartitionsPut = "SESSION_PARTITIONS_PUT"
)

var (
	ErrSessionIDRequired                  = errors.New("session id is required")
	ErrSessionStoreRequired               = errors.New("session store is not configured")
	ErrTaskStoreRequired                  = errors.New("task store is not configured")
	ErrUnsupportedSidebarPartitionVersion = errors.New("unsupported session sidebar partition version")
	ErrInvalidSessionPageQuery            = errors.New("invalid session page query")
	ErrInvalidSessionSearchQuery          = errors.New("invalid session search query")
)

type Store interface {
	ListMetadata() ([]session.SessionMetadata, error)
	SearchMetadata(string, int) ([]session.SessionMetadata, error)
	Load(string) (*session.Session, error)
	LoadPage(string, session.PageParams) (*session.Session, session.MessagePage, error)
	Save(*session.Session) error
	Delete(string) error
	LoadSidebarPartitionState() (session.SessionSidebarPartitionState, error)
	SaveSidebarPartitionState(session.SessionSidebarPartitionState) (session.SessionSidebarPartitionState, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type Service struct {
	Store         Store
	TaskStore     TaskStore
	ArtifactStore ArtifactStore
	Logger        Logger
}

func (s Service) List(traceID string) ([]api.SessionMetadata, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionList, "error", err)
		return nil, err
	}

	summaries, err := store.ListMetadata()
	if err != nil {
		s.log(traceID, ActionList, "error", err)
		return nil, err
	}

	metadata := make([]api.SessionMetadata, 0, len(summaries))
	for _, summary := range summaries {
		metadata = append(metadata, sessionturn.BuildSessionMetadataPayload(BuildMetadataInput(summary)))
	}

	s.log(traceID, ActionList, "success", nil)
	return metadata, nil
}

func (s Service) Search(query string, limit int, traceID string) ([]api.SessionMetadata, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionSearch, "error", err)
		return nil, err
	}

	matches, err := store.SearchMetadata(query, limit)
	if err != nil {
		s.log(traceID, ActionSearch, "error", err)
		return nil, err
	}
	metadata := make([]api.SessionMetadata, 0, len(matches))
	for _, summary := range matches {
		metadata = append(metadata, sessionturn.BuildSessionMetadataPayload(BuildMetadataInput(summary)))
	}

	s.log(traceID, ActionSearch, "success", nil)
	return metadata, nil
}

func (s Service) SearchParams(params api.SessionSearchParams, traceID string) ([]api.SessionMetadata, error) {
	query, limit, err := ResolveSessionSearchParams(params)
	if err != nil {
		s.log(traceID, ActionSearch, "error", err)
		return nil, err
	}
	return s.Search(query, limit, traceID)
}

func (s Service) Get(params api.SessionGetParams, traceID string) (api.SessionDetail, error) {
	if err := validateSessionGetPageParams(params); err != nil {
		s.log(traceID, ActionGet, "error", err)
		return api.SessionDetail{}, err
	}

	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionGet, "error", err)
		return api.SessionDetail{}, err
	}

	id, err := RequireSessionID(params.ID)
	if err != nil {
		return api.SessionDetail{}, err
	}

	sess, page, err := store.LoadPage(id, buildSessionPageParams(params))
	if err != nil {
		s.log(traceID, ActionGet, "error", err)
		return api.SessionDetail{}, err
	}

	payload := sessionturn.BuildSessionDetailPayload(BuildDetailInput(sess, page), params.Before == nil)
	s.log(traceID, ActionGet, "success", nil)
	return payload, nil
}

func buildSessionPageParams(params api.SessionGetParams) session.PageParams {
	pageParams := session.PageParams{Before: params.Before}
	if params.Limit != nil {
		pageParams.Limit = *params.Limit
	}
	return pageParams
}

func (s Service) Delete(params api.SessionIDParams, traceID string) (api.SessionDeleteResponse, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionDelete, "error", err)
		return api.SessionDeleteResponse{}, err
	}

	id, err := RequireSessionID(params.ID)
	if err != nil {
		return api.SessionDeleteResponse{}, err
	}

	if err := store.Delete(id); err != nil {
		s.log(traceID, ActionDelete, "error", err)
		return api.SessionDeleteResponse{}, err
	}

	s.log(traceID, ActionDelete, "success", nil)
	return api.SessionDeleteResponse{ID: id, Deleted: true}, nil
}

func (s Service) SidebarPartitions(traceID string) (api.SessionSidebarPartitionState, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionSidebarPartitionsGet, "error", err)
		return api.SessionSidebarPartitionState{}, err
	}

	state, err := store.LoadSidebarPartitionState()
	if err != nil {
		s.log(traceID, ActionSidebarPartitionsGet, "error", err)
		return api.SessionSidebarPartitionState{}, err
	}

	s.log(traceID, ActionSidebarPartitionsGet, "success", nil)
	return BuildSidebarPartitionStatePayload(state), nil
}

func (s Service) SaveSidebarPartitions(
	req api.SessionSidebarPartitionPutRequest,
	traceID string,
) (api.SessionSidebarPartitionState, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		s.log(traceID, ActionSidebarPartitionsPut, "error", err)
		return api.SessionSidebarPartitionState{}, err
	}
	if err := RequireSidebarPartitionVersion(req.Version); err != nil {
		return api.SessionSidebarPartitionState{}, err
	}

	state, err := store.SaveSidebarPartitionState(BuildSidebarPartitionStateInput(req))
	if err != nil {
		s.log(traceID, ActionSidebarPartitionsPut, "error", err)
		return api.SessionSidebarPartitionState{}, err
	}

	s.log(traceID, ActionSidebarPartitionsPut, "success", nil)
	return BuildSidebarPartitionStatePayload(state), nil
}

func RequireSessionID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", ErrSessionIDRequired
	}
	return trimmed, nil
}

func (s Service) requireSessionStore() (Store, error) {
	if s.Store == nil {
		return nil, ErrSessionStoreRequired
	}
	return s.Store, nil
}

func (s Service) taskStore() TaskStore {
	return s.TaskStore
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}
