package tasks

import "ghost-os/bridge/taskdefs"

type TaskRuntimeOverrides = taskdefs.TaskRuntimeOverrides

func CloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	return taskdefs.CloneTaskRuntimeOverrides(input)
}

func CloneActionParams(input map[string]any) map[string]any {
	return taskdefs.CloneActionParams(input)
}

func DecodeParamsMap[T any](input map[string]any) (T, error) {
	return taskdefs.DecodeParamsMap[T](input)
}

func cloneJSONValue(input any) any {
	return taskdefs.CloneJSONValue(input)
}
