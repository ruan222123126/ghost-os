import type { ButtonHTMLAttributes } from 'react';

const CLOSE_BUTTON_CLASS_NAME = 'inline-grid place-items-center rounded-full p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600';
const CLOSE_ICON_SIZE = 20;

interface CloseButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'children'> {
  'aria-label': string;
}

export function CloseButton(props: CloseButtonProps) {
  const { className, type = 'button', ...buttonProps } = props;
  const mergedClassName = [CLOSE_BUTTON_CLASS_NAME, className].filter(Boolean).join(' ');

  return (
    <button type={type} className={mergedClassName} {...buttonProps}>
      <CloseIcon />
    </button>
  );
}

function CloseIcon() {
  return (
    <svg width={CLOSE_ICON_SIZE} height={CLOSE_ICON_SIZE} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <path d="M5 5 15 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
      <path d="M15 5 5 15" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" />
    </svg>
  );
}
