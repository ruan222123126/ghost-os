interface SidebarIconProps {
  size?: number;
}

export function IconPanelLeftClose(props: SidebarIconProps) {
  const { size = 20 } = props;

  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M12.7 7.2l-2.5 2.8 2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function IconPanelLeftOpen(props: SidebarIconProps) {
  const { size = 20 } = props;

  return (
    <svg width={size} height={size} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M3.5 4.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M3.5 15.5h13" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M8.2 4.5v11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M10.2 7.2l2.5 2.8-2.5 2.8" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function IconSettings(props: SidebarIconProps) {
  const { size = 18 } = props;

  return (
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
}
