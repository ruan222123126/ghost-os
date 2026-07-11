interface TopLoadingBarProps {
  className?: string;
  label: string;
}

export function TopLoadingBar({ className, label }: TopLoadingBarProps) {
  const rootClassName = className ? `top-loading-bar ${className}` : 'top-loading-bar';

  return (
    <div className={rootClassName} role="status" aria-label={label}>
      <div className="top-loading-bar-fill" aria-hidden="true" />
    </div>
  );
}
