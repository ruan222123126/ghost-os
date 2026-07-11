import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useSessionSidebarHistoryUI } from './useSessionSidebarHistoryUI';

describe('hooks/useSessionSidebarHistoryUI', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    act(() => {
      jest.runOnlyPendingTimers();
    });
    jest.useRealTimers();
  });

  it('opens partition context menu and creates a partition', () => {
    const addPartition = jest.fn().mockReturnValue({ ok: true, createdName: 'Work' });
    const latest = renderHistoryUI({ addPartition });
    const target = buildPartitionTarget('partition-1', 'Inbox');
    const event = createMouseEvent(target, 12, 24);

    act(() => {
      latest.current.onOpenContextMenu(event);
    });

    expect(event.preventDefault).toHaveBeenCalled();
    expect(latest.current.contextMenu).toEqual({
      x: 12,
      y: 24,
      partitionID: 'partition-1',
      partitionName: 'Inbox',
    });

    act(() => {
      latest.current.openCreateDialog();
      latest.current.onChangePartitionName('Work');
    });

    act(() => {
      latest.current.onCreatePartition();
    });

    expect(addPartition).toHaveBeenCalledWith('Work');
    expect(latest.current.createDialogOpen).toBe(false);
    expect(latest.current.partitionNameInput).toBe('');
    expect(latest.current.createError).toBe('');
    expect(latest.current.statusMessage).toBe('created Work');
  });

  it('moves dragged sessions to the resolved drop index', () => {
    const moveSession = jest.fn();
    const latest = renderHistoryUI({ moveSession });
    const startEvent = createDragEvent();

    act(() => {
      latest.current.onDragStartSession(startEvent, 'session-1');
    });

    expect(startEvent.dataTransfer.effectAllowed).toBe('move');
    expect(startEvent.dataTransfer.setData).toHaveBeenCalledWith(
      'application/x-ghost-session-id',
      'session-1',
    );
    expect(latest.current.dragState.draggingSessionID).toBe('session-1');

    const overEvent = createDragEvent({ clientY: 140, rectTop: 100, rectHeight: 60 });
    act(() => {
      latest.current.onDragOverSession(overEvent, 'partition-a', 2);
    });

    expect(overEvent.preventDefault).toHaveBeenCalled();
    expect(overEvent.stopPropagation).toHaveBeenCalled();
    expect(latest.current.dragState.dropTarget).toEqual({
      partitionID: 'partition-a',
      insertIndex: 3,
    });

    const dropEvent = createDragEvent();
    act(() => {
      latest.current.onDropPartition(dropEvent, 'partition-a', 8);
    });

    expect(dropEvent.preventDefault).toHaveBeenCalled();
    expect(moveSession).toHaveBeenCalledWith('session-1', 'partition-a', 3);
    expect(latest.current.dragState.draggingSessionID).toBe('');
    expect(latest.current.dragState.dropTarget).toBeUndefined();
  });
});

function renderHistoryUI(overrides: Partial<HistoryUIOptions> = {}) {
  const latest: { current: ReturnType<typeof useSessionSidebarHistoryUI> } = {
    current: null as unknown as ReturnType<typeof useSessionSidebarHistoryUI>,
  };
  const options: HistoryUIOptions = {
    isOpen: true,
    addPartition: jest.fn().mockReturnValue({ ok: true, createdName: 'Default' }),
    moveSession: jest.fn(),
    createSuccessText: (name) => `created ${name}`,
    createErrorText: (error) => `error ${error ?? 'unknown'}`,
    ...overrides,
  };

  act(() => {
    TestRenderer.create(
      React.createElement(HistoryUIProbe, {
        options,
        onRender: (state) => {
          latest.current = state;
        },
      }),
    );
  });

  return latest;
}

function HistoryUIProbe(props: {
  options: HistoryUIOptions;
  onRender: (state: ReturnType<typeof useSessionSidebarHistoryUI>) => void;
}) {
  const state = useSessionSidebarHistoryUI(props.options);
  props.onRender(state);
  return null;
}

function buildPartitionTarget(partitionID: string, partitionName: string): HTMLElement {
  const partition = {
    dataset: {
      partitionId: partitionID,
      partitionName,
    },
  };
  return {
    closest: (selector: string) => {
      if (selector === '[data-partition-item="true"]') {
        return partition;
      }
      return null;
    },
  } as unknown as HTMLElement;
}

function createMouseEvent(
  target: HTMLElement,
  clientX: number,
  clientY: number,
): React.MouseEvent<HTMLElement> & { preventDefault: jest.Mock } {
  return {
    target,
    clientX,
    clientY,
    preventDefault: jest.fn(),
  } as unknown as React.MouseEvent<HTMLElement> & { preventDefault: jest.Mock };
}

function createDragEvent(options: {
  clientY?: number;
  rectHeight?: number;
  rectTop?: number;
} = {}): TestDragEvent {
  const dataTransfer: TestDataTransfer = {
    effectAllowed: 'none',
    setData: jest.fn(),
    getData: jest.fn().mockReturnValue(''),
  };
  return {
    clientY: options.clientY ?? 0,
    currentTarget: {
      getBoundingClientRect: () => ({
        top: options.rectTop ?? 0,
        height: options.rectHeight ?? 0,
      }),
    },
    dataTransfer,
    preventDefault: jest.fn(),
    stopPropagation: jest.fn(),
  } as unknown as TestDragEvent;
}

interface HistoryUIOptions {
  isOpen: boolean;
  addPartition: (name: string) => { ok: boolean; error?: 'empty' | 'duplicate'; createdName?: string };
  moveSession: (sessionID: string, partitionID: string, index: number) => void;
  createSuccessText: (name: string) => string;
  createErrorText: (error?: 'empty' | 'duplicate') => string;
}

interface TestDataTransfer {
  effectAllowed: string;
  getData: jest.MockedFunction<(format: string) => string>;
  setData: jest.MockedFunction<(format: string, data: string) => void>;
}

type TestDragEvent = React.DragEvent<HTMLDivElement> & {
  dataTransfer: TestDataTransfer;
  preventDefault: jest.Mock;
  stopPropagation: jest.Mock;
};
