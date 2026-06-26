import {
  ArrowDown,
  ArrowUp,
  Camera,
  Check,
  ChevronDown,
  Eye,
  FileText,
  Image as ImageIcon,
  Lightbulb,
  Menu,
  MoreVertical,
  Paperclip,
  Pencil,
  Pin,
  PinOff,
  Plus,
  Puzzle,
  Search,
  Settings,
  Sparkles,
  Square,
  SquarePen,
  Terminal,
  Trash2,
  X,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { UiIconName } from "./types";

const ICONS: Record<UiIconName, LucideIcon> = {
  "arrow-down": ArrowDown,
  "arrow-up": ArrowUp,
  camera: Camera,
  check: Check,
  "chevron-down": ChevronDown,
  edit: SquarePen,
  eye: Eye,
  "file-text": FileText,
  image: ImageIcon,
  lightbulb: Lightbulb,
  menu: Menu,
  more: MoreVertical,
  paperclip: Paperclip,
  pencil: Pencil,
  pin: Pin,
  "pin-off": PinOff,
  plus: Plus,
  puzzle: Puzzle,
  search: Search,
  settings: Settings,
  sparkles: Sparkles,
  stop: Square,
  terminal: Terminal,
  trash: Trash2,
  x: X,
};

interface IconButtonProps {
  label: string;
  icon: UiIconName;
  onClick?: () => void;
  disabled?: boolean;
  variant?: "header" | "composer";
}

export function IconButton({ label, icon, onClick, disabled = false, variant = "header" }: IconButtonProps) {
  return (
    <button
      className={`icon-button icon-button-${variant}`}
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-label={label}
      title={label}
    >
      <UiIcon name={icon} />
    </button>
  );
}

export function UiIcon(props: { name: UiIconName }) {
  const Icon = ICONS[props.name];

  return <Icon className={`ui-icon ui-icon-${props.name}`} aria-hidden="true" strokeWidth={1.75} />;
}
