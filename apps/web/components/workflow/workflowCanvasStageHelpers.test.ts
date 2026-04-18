import {
  panViewportByWheel,
  pointerToWorldPoint,
  zoomViewport,
  type CanvasViewport,
} from '@/components/workflow/workflowCanvasViewport';

describe('components/workflow/workflowCanvasStageHelpers', () => {
  it('maps pointer from screen to world coordinates', () => {
    const viewport: CanvasViewport = {
      scale: 1.5,
      offsetX: 120,
      offsetY: -40,
    };
    const point = pointerToWorldPoint({
      clientX: 340,
      clientY: 170,
      rect: { left: 20, top: 10 },
      viewport,
    });

    expect(point.x).toBeCloseTo(133.3333, 3);
    expect(point.y).toBeCloseTo(133.3333, 3);
  });

  it('keeps cursor anchored while zooming', () => {
    const viewport: CanvasViewport = {
      scale: 1,
      offsetX: 120,
      offsetY: 80,
    };
    const rect = { left: 10, top: 20 };
    const clientX = 230;
    const clientY = 170;

    const before = pointerToWorldPoint({
      clientX,
      clientY,
      rect,
      viewport,
    });
    const zoomed = zoomViewport({
      viewport,
      clientX,
      clientY,
      rect,
      deltaY: -120,
    });
    const after = pointerToWorldPoint({
      clientX,
      clientY,
      rect,
      viewport: zoomed,
    });

    expect(after.x).toBeCloseTo(before.x, 6);
    expect(after.y).toBeCloseTo(before.y, 6);
  });

  it('clamps zoom scale', () => {
    const viewport: CanvasViewport = {
      scale: 1,
      offsetX: 0,
      offsetY: 0,
    };
    const rect = { left: 0, top: 0 };

    const zoomOut = zoomViewport({
      viewport,
      clientX: 0,
      clientY: 0,
      rect,
      deltaY: 100000,
    });
    const zoomIn = zoomViewport({
      viewport,
      clientX: 0,
      clientY: 0,
      rect,
      deltaY: -100000,
    });

    expect(zoomOut.scale).toBe(0.4);
    expect(zoomIn.scale).toBe(2.5);
  });

  it('pans viewport by wheel delta', () => {
    const viewport: CanvasViewport = {
      scale: 1,
      offsetX: 30,
      offsetY: -12,
    };
    const next = panViewportByWheel(viewport, 8, -20);

    expect(next.offsetX).toBe(22);
    expect(next.offsetY).toBe(8);
    expect(next.scale).toBe(1);
  });
});
