import type { ReactNode } from 'react';

interface TaskFormFieldProps {
  label: string;
  description: string;
  children: ReactNode;
}

export function TaskFormField(props: TaskFormFieldProps) {
  const { label, description, children } = props;

  return (
    <label className="block">
      <span className="mb-1 block text-[13px] font-medium text-[#111111]">{label}</span>
      <span className="mb-2 block text-[12px] text-[#737373]">{description}</span>
      {children}
    </label>
  );
}
