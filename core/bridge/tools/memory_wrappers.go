package tools

import (
	memoryaug "ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	toolmemory "ghost-os/bridge/tools/memory"
)

type MemoryManageTool = toolmemory.MemoryManageTool

type MemoryLearnedListTool = toolmemory.MemoryLearnedListTool

type MemoryRecallDebugTool = toolmemory.MemoryRecallDebugTool

func NewMemoryManageTool(store *memorystore.Store) *MemoryManageTool {
	return toolmemory.NewMemoryManageTool(store)
}

func NewMemoryManageToolFromEnv() (*MemoryManageTool, error) {
	return toolmemory.NewMemoryManageToolFromEnv()
}

func NewMemoryLearnedListTool(store *memorystore.Store) *MemoryLearnedListTool {
	return toolmemory.NewMemoryLearnedListTool(store)
}

func NewMemoryRecallDebugTool(
	planner memoryaug.IntentPlanner,
	service memoryaug.RecallService,
	userScopeID string,
) *MemoryRecallDebugTool {
	return toolmemory.NewMemoryRecallDebugTool(planner, service, userScopeID)
}
