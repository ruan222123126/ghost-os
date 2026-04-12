import type { FC } from 'react';

export const IconPanelLeftClose: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M12.7 7.2l-2.5 2.8 2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

export const IconPanelLeftOpen: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M10.2 7.2l2.5 2.8-2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

export const IconSearch: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M8.9 14.7a5.8 5.8 0 1 1 0-11.6 5.8 5.8 0 0 1 0 11.6Z" stroke="currentColor" strokeWidth="1.5" />
    <path d="M13.3 13.3 17 17" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
  </svg>
);

export const IconPlus: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M10 4v12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    <path d="M4 10h12" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
  </svg>
);

export const IconX: FC<{ size?: number }> = ({ size = 14 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path d="M5 5l10 10" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    <path d="M15 5 5 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
  </svg>
);
