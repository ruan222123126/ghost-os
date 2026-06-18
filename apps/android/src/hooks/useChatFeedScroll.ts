import { useEffect, useRef, useState } from "react";

const SCROLL_DOWN_THRESHOLD_PX = 50;

export function useChatFeedScroll(lastUserMessage: string, reply: unknown) {
  const scrollRef = useRef<HTMLElement>(null);
  const [showScrollDown, setShowScrollDown] = useState(false);

  useEffect(() => {
    if (lastUserMessage || reply) {
      scrollToBottom("smooth");
    }
  }, [lastUserMessage, reply]);

  function handleScroll(): void {
    if (scrollRef.current) {
      setShowScrollDown(shouldShowScrollDown(scrollRef.current));
    }
  }

  function resetScrollDown(): void {
    setShowScrollDown(false);
  }

  function scrollToBottom(behavior: ScrollBehavior = "smooth"): void {
    setShowScrollDown(false);
    if (scrollRef.current) {
      scrollRef.current.scrollTo({
        top: scrollRef.current.scrollHeight,
        behavior,
      });
    }
  }

  return {
    handleScroll,
    resetScrollDown,
    scrollRef,
    scrollToBottom,
    showScrollDown,
  };
}

function shouldShowScrollDown(element: HTMLElement): boolean {
  return element.scrollHeight - element.scrollTop - element.clientHeight > SCROLL_DOWN_THRESHOLD_PX;
}
