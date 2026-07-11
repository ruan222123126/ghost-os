import { UiIcon } from "./icons";
import type { UiIconName } from "./types";
import "./MoreActionSheet.css";

interface MoreActionSheetProps {
  open: boolean;
  hasLocalConversation: boolean;
  canTogglePin: boolean;
  isPinned: boolean;
  onClose: () => void;
  onTogglePin: () => void;
  onClearConversation: () => void;
}

export function MoreActionSheet(props: MoreActionSheetProps) {
  return (
    <>
      <button
        className={`sheet-scrim ${props.open ? "is-open" : ""}`}
        type="button"
        aria-label="关闭更多操作"
        onClick={props.onClose}
      />
      <div
        className={`action-sheet ${props.open ? "is-open" : ""}`}
        role="dialog"
        aria-hidden={!props.open}
        aria-modal="true"
        inert={props.open ? undefined : true}
      >
        <div className="sheet-handle" aria-hidden="true" />
        <div className="action-sheet-list">
          <ActionSheetButton
            icon={props.isPinned ? "pin-off" : "pin"}
            label={props.isPinned ? "取消固定" : "固定"}
            disabled={!props.canTogglePin}
            onClick={props.onTogglePin}
          />
          <ActionSheetButton icon="pencil" label="重命名" onClick={props.onClose} />
          <ActionSheetButton
            icon="trash"
            label="删除"
            disabled={!props.hasLocalConversation}
            onClick={props.onClearConversation}
          />
        </div>
      </div>
    </>
  );
}

function ActionSheetButton(props: { icon: UiIconName; label: string; disabled?: boolean; onClick: () => void }) {
  return (
    <button className="action-sheet-button" type="button" disabled={props.disabled} onClick={props.onClick}>
      <UiIcon name={props.icon} />
      <span>{props.label}</span>
    </button>
  );
}
