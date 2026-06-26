'use client';

import type { FC, MouseEvent } from 'react';
import { useWebLocale } from '@/lib/i18n/provider';

interface SidebarSettingsButtonProps {
  collapsed: boolean;
  onClick: () => void;
}

const IconSettings: FC<{ size?: number }> = ({ size = 20 }) => (
  <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
    <path
      d="M10 3.25 11.35 2.5l1.1 1.9 1.6.4 1.55-1.15 1.4 1.4-1.15 1.55.4 1.6 1.9 1.1-.75 1.35.75 1.35-1.9 1.1-.4 1.6 1.15 1.55-1.4 1.4-1.55-1.15-1.6.4-1.1 1.9L10 16.75l-1.35.75-1.1-1.9-1.6-.4-1.55 1.15-1.4-1.4 1.15-1.55-.4-1.6-1.9-1.1.75-1.35-.75-1.35 1.9-1.1.4-1.6L3 5.05l1.4-1.4 1.55 1.15 1.6-.4 1.1-1.9L10 3.25Z"
      stroke="currentColor"
      strokeWidth="1.2"
      strokeLinejoin="round"
    />
    <circle cx="10" cy="10" r="2.6" stroke="currentColor" strokeWidth="1.3" />
  </svg>
);

export const SidebarSettingsButton: FC<SidebarSettingsButtonProps> = ({ collapsed, onClick }) => (
  <ButtonContent collapsed={collapsed} onClick={onClick} />
);

function ButtonContent(props: SidebarSettingsButtonProps) {
  const { collapsed, onClick } = props;
  const { copy } = useWebLocale();
  const handleClick = (event: MouseEvent<HTMLButtonElement>) => {
    updateSettingsPanelOrigin(event.currentTarget);
    onClick();
  };

  return (
    <div className="mt-auto border-t border-black/10 px-3 py-3">
      <button
        type="button"
        onClick={handleClick}
        className={`group flex items-center justify-center gap-2 overflow-hidden bg-white text-black transition-colors hover:bg-black hover:text-white ${
          collapsed ? 'mx-auto h-10 w-10' : 'w-full px-4 py-3'
        }`}
        aria-label={copy.chat.sidebarOpenSettingsAria}
        aria-haspopup="dialog"
        title={copy.chat.sidebarSettingsTitle}
      >
        <span className="grid flex-shrink-0 place-items-center">
          <IconSettings />
        </span>
        <span
          className={`whitespace-nowrap text-xs font-bold uppercase tracking-tighter text-neutral-500 transition-all duration-300 group-hover:text-white ${
            collapsed ? 'max-w-0 opacity-0' : 'max-w-[100px] opacity-100'
          }`}
        >
          {copy.chat.sidebarSettingsTitle}
        </span>
      </button>
    </div>
  );
}

function updateSettingsPanelOrigin(trigger: HTMLButtonElement) {
  if (typeof document === 'undefined') {
    return;
  }

  const rect = trigger.getBoundingClientRect();
  document.documentElement.style.setProperty('--settings-panel-origin-x', `${rect.left + rect.width / 2}px`);
  document.documentElement.style.setProperty('--settings-panel-origin-y', `${rect.top + rect.height / 2}px`);
}
