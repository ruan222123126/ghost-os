import type { Virtualizer } from '@tanstack/react-virtual';

export interface PrependAnchor {
  firstVisibleCommittedMessageId: string | null;
  loadingOffsetPx: number;
  rowKey: string | null;
  rowOffsetPx: number;
  settling: boolean;
  scrollHeight: number;
  scrollTop: number;
  visibleCommittedMessageCount: number;
}

export function capturePrependAnchor(options: {
  container: HTMLDivElement | null;
  firstVisibleCommittedMessageId: string | null;
  rowKeys: readonly string[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  visibleCommittedMessageCount: number;
}): PrependAnchor | null {
  const {
    container,
    firstVisibleCommittedMessageId,
    rowKeys,
    rowVirtualizer,
    visibleCommittedMessageCount,
  } = options;
  if (!container) {
    return null;
  }
  const rowAnchor = captureVisibleRowAnchor({
    rowKeys,
    rowVirtualizer,
    scrollTop: container.scrollTop,
  });

  return {
    firstVisibleCommittedMessageId,
    loadingOffsetPx: 0,
    rowKey: rowAnchor?.rowKey ?? null,
    rowOffsetPx: rowAnchor?.rowOffsetPx ?? 0,
    scrollHeight: container.scrollHeight,
    scrollTop: container.scrollTop,
    settling: false,
    visibleCommittedMessageCount,
  };
}

export function restorePrependAnchorPosition(options: {
  anchor: PrependAnchor;
  container: HTMLDivElement;
  rowKeys: readonly string[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
}) {
  const restored = restorePrependAnchorByRowKey(options);
  if (restored) {
    return;
  }

  applyPrependAnchorOffset(options.container, options.anchor);
}

export function restorePrependAnchorAfterSkippedLoad(
  container: HTMLDivElement,
  anchor: PrependAnchor,
) {
  if (anchor.loadingOffsetPx === 0) {
    return;
  }

  container.scrollTop = anchor.scrollTop - anchor.loadingOffsetPx;
}

function captureVisibleRowAnchor(options: {
  rowKeys: readonly string[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
  scrollTop: number;
}): { rowKey: string; rowOffsetPx: number } | null {
  const anchorItem = options.rowVirtualizer
    .getVirtualItems()
    .find((item) => item.end > options.scrollTop && options.rowKeys[item.index]);
  if (!anchorItem) {
    return null;
  }

  return {
    rowKey: options.rowKeys[anchorItem.index],
    rowOffsetPx: anchorItem.start - options.scrollTop,
  };
}

function restorePrependAnchorByRowKey(options: {
  anchor: PrependAnchor;
  container: HTMLDivElement;
  rowKeys: readonly string[];
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>;
}): boolean {
  if (!options.anchor.rowKey) {
    return false;
  }

  const rowIndex = options.rowKeys.indexOf(options.anchor.rowKey);
  if (rowIndex < 0) {
    return false;
  }
  const rowStartPx = getVirtualRowStartPx(options.rowVirtualizer, rowIndex);
  if (rowStartPx === null) {
    return false;
  }

  options.container.scrollTop = Math.max(0, rowStartPx - options.anchor.rowOffsetPx);
  options.anchor.scrollHeight = options.container.scrollHeight;
  options.anchor.scrollTop = options.container.scrollTop;
  return true;
}

function getVirtualRowStartPx(
  rowVirtualizer: Virtualizer<HTMLDivElement, Element>,
  rowIndex: number,
): number | null {
  const measured = rowVirtualizer.measurementsCache[rowIndex];
  if (measured) {
    return measured.start;
  }

  const offset = rowVirtualizer.getOffsetForIndex(rowIndex, 'start');
  return offset?.[0] ?? null;
}

function applyPrependAnchorOffset(container: HTMLDivElement, anchor: PrependAnchor) {
  const offsetPx = container.scrollHeight - anchor.scrollHeight;
  if (offsetPx === 0) {
    return;
  }

  container.scrollTop = anchor.scrollTop + offsetPx;
  anchor.scrollHeight = container.scrollHeight;
  anchor.scrollTop = container.scrollTop;
  anchor.loadingOffsetPx += offsetPx;
}
