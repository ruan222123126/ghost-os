import type { WorkflowCanvasPosition } from '@/lib/workflow-editor';

export interface CanvasViewport {
  scale: number;
  offsetX: number;
  offsetY: number;
}

interface CanvasRect {
  left: number;
  top: number;
}

const MIN_SCALE = 0.4;
const MAX_SCALE = 2.5;
const SCALE_PER_DELTA = 0.0012;

export function pointerToWorldPoint(options: {
  clientX: number;
  clientY: number;
  rect: CanvasRect;
  viewport: CanvasViewport;
}): WorkflowCanvasPosition {
  const {
    clientX,
    clientY,
    rect,
    viewport,
  } = options;
  return {
    x: (clientX - rect.left - viewport.offsetX) / viewport.scale,
    y: (clientY - rect.top - viewport.offsetY) / viewport.scale,
  };
}

export function zoomViewport(options: {
  viewport: CanvasViewport;
  clientX: number;
  clientY: number;
  rect: CanvasRect;
  deltaY: number;
}): CanvasViewport {
  const {
    viewport,
    clientX,
    clientY,
    rect,
    deltaY,
  } = options;
  const nextScale = clampScale(viewport.scale * (1 - deltaY * SCALE_PER_DELTA));
  if (nextScale === viewport.scale) {
    return viewport;
  }
  const pointX = clientX - rect.left;
  const pointY = clientY - rect.top;
  const worldX = (pointX - viewport.offsetX) / viewport.scale;
  const worldY = (pointY - viewport.offsetY) / viewport.scale;
  return {
    scale: nextScale,
    offsetX: pointX - worldX * nextScale,
    offsetY: pointY - worldY * nextScale,
  };
}

export function panViewportByWheel(
  viewport: CanvasViewport,
  deltaX: number,
  deltaY: number,
): CanvasViewport {
  return {
    ...viewport,
    offsetX: viewport.offsetX - deltaX,
    offsetY: viewport.offsetY - deltaY,
  };
}

function clampScale(value: number): number {
  return Math.min(MAX_SCALE, Math.max(MIN_SCALE, value));
}
