import React from 'react';
import TestRenderer, { act } from 'react-test-renderer';
import { useSessionSidebarState } from './useSessionSidebarState';

describe('hooks/useSessionSidebarState', () => {
  it('keeps search state consistent when toggling search and sidebar', () => {
    let latest!: ReturnType<typeof useSessionSidebarState>;

    act(() => {
      TestRenderer.create(
        React.createElement(SessionSidebarStateProbe, {
          onRender: (state) => {
            latest = state;
          },
        }),
      );
    });

    expect(latest.isOpen).toBe(true);
    expect(latest.isSearchVisible).toBe(false);

    act(() => {
      latest.toggleSearch();
      latest.setSearchQuery('report');
    });

    expect(latest.isOpen).toBe(true);
    expect(latest.isSearchVisible).toBe(true);
    expect(latest.searchQuery).toBe('report');

    act(() => {
      latest.toggleSidebar();
    });

    expect(latest.isOpen).toBe(false);
    expect(latest.isSearchVisible).toBe(false);
    expect(latest.searchQuery).toBe('');

    act(() => {
      latest.toggleSidebar();
    });

    expect(latest.isOpen).toBe(true);
    expect(latest.isSearchVisible).toBe(false);
  });
});

function SessionSidebarStateProbe(props: {
  onRender: (state: ReturnType<typeof useSessionSidebarState>) => void;
}) {
  const state = useSessionSidebarState();
  props.onRender(state);
  return null;
}
