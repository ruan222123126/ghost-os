// Session use cases for create/list/get/update/delete operations.

package orchestration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	appdownloads "ghost-os/bridge/orchestration/internal/app/downloads"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/session"
)

var errSessionEnded = errors.New("session has already ended")

type BinaryDownload = appdownloads.BinaryDownload

// executeSessionsListAction 汇总全部会话元数据，并映射为 API 返回结构。
func (s *bridgeService) executeSessionsListAction(traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	metadata, err := usecase.List(traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(metadata), nil
}

func (s *bridgeService) executeSessionsSearchAction(params sessionSearchParams, traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	metadata, err := usecase.SearchParams(params, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(metadata), nil
}

func (s *bridgeService) executeSessionsSearchQueryAction(query string, limit int, traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	metadata, err := usecase.Search(query, limit, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(metadata), nil
}

// executeSessionGetAction 读取并返回单会话详情页。
func (s *bridgeService) executeSessionGetAction(params sessionGetParams, traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	detail, err := usecase.Get(params, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(detail), nil
}

func (s *bridgeService) executeSessionSourcesAction(traceID string) (ServiceResult, error) {
	usecase, err := s.sessionSourcesUsecase(traceID)
	if err != nil {
		return ServiceResult{}, err
	}

	resolution, err := usecase.Sources(traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(resolution), nil
}

// executeSessionDeleteAction 删除指定会话，并返回幂等友好的删除结果结构。
func (s *bridgeService) executeSessionDeleteAction(params sessionIDParams, traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	payload, err := usecase.Delete(params, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(payload), nil
}

func (s *bridgeService) executeSessionSidebarPartitionsGetAction(traceID string) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	payload, err := usecase.SidebarPartitions(traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(payload), nil
}

func (s *bridgeService) executeSessionSidebarPartitionsPutAction(
	req sessionSidebarPartitionPutRequest,
	traceID string,
) (ServiceResult, error) {
	usecase, err := s.sessionUsecase()
	if err != nil {
		return ServiceResult{}, err
	}

	payload, err := usecase.SaveSidebarPartitions(req, traceID)
	if err != nil {
		return ServiceResult{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return bus.ResultSuccess(payload), nil
}

func (s *Service) ExecuteSessionSidebarPartitionsGetAction(traceID string) (ServiceResult, error) {
	return s.inner.executeSessionSidebarPartitionsGetAction(traceID)
}

func (s *Service) ExecuteSessionSidebarPartitionsPutAction(
	req SessionSidebarPartitionPutRequest,
	traceID string,
) (ServiceResult, error) {
	return s.inner.executeSessionSidebarPartitionsPutAction(req, traceID)
}

func (s *Service) OpenSessionArtifactDownload(sessionID string, artifactID string) (BinaryDownload, error) {
	usecase, err := s.inner.sessionArtifactUsecase()
	if err != nil {
		return BinaryDownload{}, err
	}
	download, err := usecase.DownloadArtifact(sessionID, artifactID, "")
	if err != nil {
		return BinaryDownload{}, bus.WrapError(mapSessionAppErrorKind(err), err)
	}
	return BinaryDownload{
		Reader:   download.Reader,
		Name:     download.Name,
		MimeType: download.MimeType,
		Size:     download.Size,
		SHA256:   download.SHA256,
	}, nil
}

func (s *bridgeService) sessionUsecase() (appsessions.Service, error) {
	store, err := s.requireSessionStore()
	if err != nil {
		return appsessions.Service{}, err
	}
	return appsessions.Service{
		Store:  store,
		Logger: serviceActionLogger{},
	}, nil
}

func (s *bridgeService) sessionSourcesUsecase(traceID string) (appsessions.Service, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		logAction(traceID, appsessions.ActionSources, "error", err)
		return appsessions.Service{}, bus.WrapError(bus.ErrorKindFromStatus(code), err)
	}
	return appsessions.Service{
		TaskStore: store,
		Logger:    serviceActionLogger{},
	}, nil
}

func (s *bridgeService) sessionArtifactUsecase() (appsessions.Service, error) {
	if s.artifactInitErr != nil {
		return appsessions.Service{}, bus.WrapError(ServiceErrorInternal, s.artifactInitErr)
	}
	return appsessions.Service{
		ArtifactStore: s.artifactStore,
		Logger:        serviceActionLogger{},
	}, nil
}

func mapSessionAppErrorKind(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, appsessions.ErrSessionIDRequired),
		errors.Is(err, appsessions.ErrInvalidSessionSearchQuery),
		errors.Is(err, appsessions.ErrUnsupportedSidebarPartitionVersion),
		errors.Is(err, appsessions.ErrInvalidSessionPageQuery),
		errors.Is(err, appsessions.ErrArtifactIDRequired),
		errors.Is(err, appsessions.ErrInvalidArtifactSessionID),
		errors.Is(err, appsessions.ErrInvalidArtifactID):
		return ServiceErrorInvalidInput
	case errors.Is(err, appsessions.ErrArtifactNotFound):
		return ServiceErrorNotFound
	case errors.Is(err, appsessions.ErrSessionStoreRequired),
		errors.Is(err, appsessions.ErrTaskStoreRequired),
		errors.Is(err, appsessions.ErrArtifactStoreRequired),
		errors.Is(err, appsessions.ErrInvalidArtifactPath):
		return ServiceErrorInternal
	default:
		return mapSessionStorageErrorKind(err)
	}
}

// requireSessionStore 确保当前 service 已配置持久化会话存储。
func (s *bridgeService) requireSessionStore() (*session.Store, error) {
	if s.sessionStore == nil {
		return nil, bus.WrapError(ServiceErrorInternal, errors.New("session store is not configured"))
	}
	return s.sessionStore, nil
}

// requireSessionID 对输入 id 做最小合法性校验并返回 trim 后值。
func requireSessionID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", bus.WrapError(ServiceErrorInvalidInput, errors.New("session id is required"))
	}
	return trimmed, nil
}

func mapSessionStorageErrorKind(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return ServiceErrorInvalidInput
	case errors.Is(err, session.ErrSessionNotFound):
		return ServiceErrorNotFound
	case errors.Is(err, errSessionEnded):
		return ServiceErrorConflict
	default:
		return ServiceErrorInternal
	}
}

// ensureSessionActive 在继续已有会话前校验其可续跑状态。
func (s *bridgeService) ensureSessionActive(sessionID string) error {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil
	}
	if s == nil || s.sessionStore == nil {
		return bus.WrapError(ServiceErrorInternal, errors.New("session store is not configured"))
	}

	sess, err := s.sessionStore.Load(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return bus.WrapError(
				ServiceErrorNotFound,
				fmt.Errorf("%w: session_id=%s (omit session_id to start a new session)", session.ErrSessionNotFound, id),
			)
		}
		return bus.WrapError(mapSessionStorageErrorKind(err), err)
	}
	if !sess.IsEnded() {
		return nil
	}
	return bus.WrapError(ServiceErrorConflict, fmt.Errorf("%w: session_id=%s", errSessionEnded, id))
}

// markSessionEnded 在收到结构化结束信号后把会话状态持久化为 ended。
func (s *bridgeService) markSessionEnded(sessionID string) error {
	store := s.sessionStore
	if store == nil {
		return bus.WrapError(ServiceErrorInternal, errors.New("session store is not configured"))
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return bus.WrapError(ServiceErrorInternal, errors.New("session id is empty"))
	}

	sess, err := store.Load(id)
	if err != nil {
		return bus.WrapError(mapSessionStorageErrorKind(err), err)
	}
	if sess.IsEnded() {
		return nil
	}

	sess.MarkEnded(time.Now().UTC())
	if err := store.Save(sess); err != nil {
		return bus.WrapError(mapSessionStorageErrorKind(err), err)
	}
	return nil
}
