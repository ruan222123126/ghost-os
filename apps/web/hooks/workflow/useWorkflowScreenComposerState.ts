'use client';

import { useEffect, useState, type Dispatch, type SetStateAction } from 'react';
import {
  appendScreenControlComposerStep,
  moveScreenControlComposerStep,
  removeScreenControlComposerStep,
  syncScreenControlComposerStepsToToolArguments,
  updateScreenControlComposerStep,
  type ScreenControlAtomicAction,
  type ScreenControlComposerStep,
  type WorkflowCanvasNodeDraft,
  withScreenControlComposerSteps,
} from '@/lib/workflow-editor';

interface UseWorkflowScreenComposerStateOptions {
  selectedNode: WorkflowCanvasNodeDraft;
  selectedToolName: string;
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void;
}

export interface WorkflowScreenComposerState {
  open: boolean;
  steps: ScreenControlComposerStep[];
  isScreenControlTool: boolean;
  onOpen: () => void;
  onClose: () => void;
  onAppend: (action: ScreenControlAtomicAction) => void;
  onMove: (index: number, direction: 'up' | 'down') => void;
  onRemove: (index: number) => void;
  onUpdateStep: (index: number, step: ScreenControlComposerStep) => void;
}

export function useWorkflowScreenComposerState(
  options: UseWorkflowScreenComposerStateOptions,
): WorkflowScreenComposerState {
  const { selectedNode, selectedToolName, onUpdateNode } = options;
  const [open, setOpen] = useState(false);
  const isScreenControlTool = isScreenControlToolName(selectedToolName);
  const steps = selectedNode.ui.screenControlComposer?.steps ?? [];
  const updateSteps = createScreenComposerStepUpdater(selectedNode, onUpdateNode);

  useCloseComposerForNonScreenControlTool(isScreenControlTool, setOpen);

  return createScreenComposerState({
    isScreenControlTool,
    open,
    setOpen,
    steps,
    updateSteps,
  });
}

function createScreenComposerState(options: {
  isScreenControlTool: boolean;
  open: boolean;
  setOpen: Dispatch<SetStateAction<boolean>>;
  steps: ScreenControlComposerStep[];
  updateSteps: (steps: ScreenControlComposerStep[]) => void;
}): WorkflowScreenComposerState {
  const { isScreenControlTool, open, setOpen, steps, updateSteps } = options;
  return {
    open: isScreenControlTool && open,
    steps,
    isScreenControlTool,
    onOpen: createComposerOpenHandler(setOpen),
    onClose: createComposerCloseHandler(setOpen),
    onAppend: createAppendStepHandler(steps, updateSteps),
    onMove: createMoveStepHandler(steps, updateSteps),
    onRemove: createRemoveStepHandler(steps, updateSteps),
    onUpdateStep: createUpdateStepHandler(steps, updateSteps),
  };
}

function createComposerOpenHandler(setOpen: Dispatch<SetStateAction<boolean>>): () => void {
  return function openComposer() {
    setOpen(true);
  };
}

function createComposerCloseHandler(setOpen: Dispatch<SetStateAction<boolean>>): () => void {
  return function closeComposer() {
    setOpen(false);
  };
}

function createAppendStepHandler(
  steps: ScreenControlComposerStep[],
  updateSteps: (steps: ScreenControlComposerStep[]) => void,
): (action: ScreenControlAtomicAction) => void {
  return function appendStep(action) {
    updateSteps(appendScreenControlComposerStep(steps, action));
  };
}

function createMoveStepHandler(
  steps: ScreenControlComposerStep[],
  updateSteps: (steps: ScreenControlComposerStep[]) => void,
): (index: number, direction: 'up' | 'down') => void {
  return function moveStep(index, direction) {
    updateSteps(moveScreenControlComposerStep(steps, index, direction));
  };
}

function createRemoveStepHandler(
  steps: ScreenControlComposerStep[],
  updateSteps: (steps: ScreenControlComposerStep[]) => void,
): (index: number) => void {
  return function removeStep(index) {
    updateSteps(removeScreenControlComposerStep(steps, index));
  };
}

function createUpdateStepHandler(
  steps: ScreenControlComposerStep[],
  updateSteps: (steps: ScreenControlComposerStep[]) => void,
): (index: number, step: ScreenControlComposerStep) => void {
  return function updateStep(index, step) {
    updateSteps(updateScreenControlComposerStep(steps, index, step));
  };
}

function createScreenComposerStepUpdater(
  selectedNode: WorkflowCanvasNodeDraft,
  onUpdateNode: (node: WorkflowCanvasNodeDraft) => void,
): (steps: ScreenControlComposerStep[]) => void {
  return function updateSteps(nextSteps) {
    const nodeWithSteps = withScreenControlComposerSteps(selectedNode, nextSteps);
    onUpdateNode(syncScreenControlComposerStepsToToolArguments(nodeWithSteps, nextSteps));
  };
}

function useCloseComposerForNonScreenControlTool(
  isScreenControlTool: boolean,
  setOpen: Dispatch<SetStateAction<boolean>>,
): void {
  useEffect(() => {
    if (isScreenControlTool) {
      return;
    }
    setOpen(false);
  }, [isScreenControlTool, setOpen]);
}

function isScreenControlToolName(toolName: string): boolean {
  return toolName.trim() === 'screen_control';
}
