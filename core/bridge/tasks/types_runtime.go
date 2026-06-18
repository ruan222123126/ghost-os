package tasks

import "ghost-os/bridge/taskdefs"

type TaskRuntimeOverrides = taskdefs.TaskRuntimeOverrides

func CloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	return taskdefs.CloneTaskRuntimeOverrides(input)
}

func cloneJSONValue(input any) any {
	return taskdefs.CloneJSONValue(input)
}
