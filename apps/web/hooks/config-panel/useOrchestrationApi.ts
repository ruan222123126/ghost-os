'use client';

import { useMemo } from 'react';
import {
  createOrchestration as createOrchestrationRequest,
  deleteOrchestration as deleteOrchestrationRequest,
  listOrchestrations as listOrchestrationsRequest,
  runOrchestrationNow as runOrchestrationNowRequest,
  updateOrchestration as updateOrchestrationRequest,
} from '@/lib/api/orchestrations/api';
import { migrateLegacyOrchestrations as migrateLegacyOrchestrationsRequest } from '@/lib/orchestration-editor/legacyMigration';

export function useOrchestrationApi() {
  return useMemo(() => ({
    migrateLegacyOrchestrations: migrateLegacyOrchestrationsRequest,
    listOrchestrations: listOrchestrationsRequest,
    createOrchestration: createOrchestrationRequest,
    updateOrchestration: updateOrchestrationRequest,
    deleteOrchestration: deleteOrchestrationRequest,
    runOrchestrationNow: runOrchestrationNowRequest,
  }), []);
}
