'use client';

import { useMemo } from 'react';
import {
  createTask as createTaskRequest,
  deleteTask as deleteTaskRequest,
  listTasks as listTasksRequest,
  runTaskNow as runTaskNowRequest,
  updateTask as updateTaskRequest,
} from '@/lib/api/tasks/api';

export function useTasksApi() {
  return useMemo(() => ({
    listTasks: listTasksRequest,
    createTask: createTaskRequest,
    updateTask: updateTaskRequest,
    deleteTask: deleteTaskRequest,
    runTaskNow: runTaskNowRequest,
  }), []);
}
