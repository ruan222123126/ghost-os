import type { CanvasViewport } from '@/components/workflow/workflowCanvasViewport';

const GRID_BASE_SIZE = 40;
const GRID_MAJOR_FACTOR = 5;
const GRID_BACKGROUND_COLOR = '#f5f5f5';
const GRID_MINOR_COLOR = 'rgb(0 0 0 / 0.06)';
const GRID_MAJOR_COLOR = 'rgb(0 0 0 / 0.12)';
const GRID_LINE_WIDTH = 1;

export function drawWorkflowCanvasBackground(
  background: HTMLCanvasElement,
  canvas: HTMLDivElement,
  viewport: CanvasViewport,
) {
  const rect = canvas.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) {
    return;
  }

  const ratio = window.devicePixelRatio || 1;
  background.width = Math.floor(rect.width * ratio);
  background.height = Math.floor(rect.height * ratio);
  background.style.width = `${rect.width}px`;
  background.style.height = `${rect.height}px`;

  const context = background.getContext('2d');
  if (!context) {
    return;
  }

  context.setTransform(ratio, 0, 0, ratio, 0, 0);
  context.clearRect(0, 0, rect.width, rect.height);
  context.fillStyle = GRID_BACKGROUND_COLOR;
  context.fillRect(0, 0, rect.width, rect.height);
  drawGridLines(context, rect.width, rect.height, viewport);
}

function drawGridLines(
  context: CanvasRenderingContext2D,
  width: number,
  height: number,
  viewport: CanvasViewport,
) {
  const minorStep = GRID_BASE_SIZE * viewport.scale;
  const majorStep = minorStep * GRID_MAJOR_FACTOR;
  drawGridAxis(context, width, height, normalizeGridOffset(viewport.offsetX, minorStep), minorStep, true, GRID_MINOR_COLOR);
  drawGridAxis(context, width, height, normalizeGridOffset(viewport.offsetY, minorStep), minorStep, false, GRID_MINOR_COLOR);
  drawGridAxis(context, width, height, normalizeGridOffset(viewport.offsetX, majorStep), majorStep, true, GRID_MAJOR_COLOR);
  drawGridAxis(context, width, height, normalizeGridOffset(viewport.offsetY, majorStep), majorStep, false, GRID_MAJOR_COLOR);
}

function drawGridAxis(
  context: CanvasRenderingContext2D,
  width: number,
  height: number,
  start: number,
  step: number,
  vertical: boolean,
  color: string,
) {
  context.beginPath();
  context.strokeStyle = color;
  context.lineWidth = GRID_LINE_WIDTH;
  for (let value = start; value <= (vertical ? width : height); value += step) {
    const fixedValue = Math.round(value) + 0.5;
    if (vertical) {
      context.moveTo(fixedValue, 0);
      context.lineTo(fixedValue, height);
      continue;
    }
    context.moveTo(0, fixedValue);
    context.lineTo(width, fixedValue);
  }
  context.stroke();
}

function normalizeGridOffset(offset: number, step: number): number {
  const remainder = offset % step;
  if (remainder >= 0) {
    return remainder;
  }
  return remainder + step;
}
