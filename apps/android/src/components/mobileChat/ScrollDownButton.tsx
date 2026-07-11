import { ArrowDown } from "lucide-react";
import "./ScrollDownButton.css";

interface ScrollDownButtonProps {
  onClick: () => void;
}

export function ScrollDownButton(props: ScrollDownButtonProps) {
  return (
    <button className="scroll-down-button" type="button" onClick={props.onClick} aria-label="滚动到底部">
      <ArrowDown className="ui-icon" aria-hidden="true" strokeWidth={1.75} />
    </button>
  );
}
