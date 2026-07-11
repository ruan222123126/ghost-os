'use client';

import { useEffect, useRef, type RefObject } from 'react';
import { drawWorkflowCanvasBackground } from '@/components/workflow/workflowCanvasBackground';
import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';

interface WorkflowCanvasBackgroundOptions {
  backgroundCanvasRef: RefObject<HTMLCanvasElement>;
  canvasRef: RefObject<HTMLDivElement>;
  viewport: CanvasViewport;
}

export function useWorkflowCanvasBackground(options: WorkflowCanvasBackgroundOptions): void {
  const { backgroundCanvasRef, canvasRef, viewport } = options;
  const viewportRef = useRef<CanvasViewport>(viewport);

  useEffect(() => {
    viewportRef.current = viewport;
    drawBackground(canvasRef.current, backgroundCanvasRef.current, viewport);
  }, [viewport]);

  useEffect(() => {
    const canvas = canvasRef.current;
    const background = backgroundCanvasRef.current;
    if (!canvas || !background) {
      return;
    }

    const observer = new ResizeObserver(() => {
      drawWorkflowCanvasBackground(background, canvas, viewportRef.current);
    });
    observer.observe(canvas);
    return () => observer.disconnect();
  }, []);
}

function drawBackground(
  canvas: HTMLDivElement | null,
  background: HTMLCanvasElement | null,
  viewport: CanvasViewport,
): void {
  if (!canvas || !background) {
    return;
  }
  drawWorkflowCanvasBackground(background, canvas, viewport);
}
