package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func clickByTextScript(text string) string {
	encoded, _ := json.Marshal(text)
	return fmt.Sprintf(`(() => {
  const query = %s;
  const candidates = Array.from(document.querySelectorAll('a,button,input,textarea,[role="button"],[role="link"]'));
  const exact = candidates.find(el => (el.innerText || el.value || '').trim() === query);
  if (exact) { exact.click(); return true; }
  const fallback = Array.from(document.querySelectorAll('*')).find(el => (el.innerText || '').trim() === query);
  if (fallback) { fallback.click(); return true; }
  return false;
})()`, string(encoded))
}

func clickByXPathScript(xpath string) string {
	encoded, _ := json.Marshal(xpath)
	return fmt.Sprintf(`(() => {
  const path = %s;
  const result = document.evaluate(path, document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null);
  const node = result.singleNodeValue;
  if (node && node.click) { node.click(); return true; }
  return false;
})()`, string(encoded))
}

func normalizeKeyPress(key string) string {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "enter":
		return "\n"
	case "tab":
		return "\t"
	case "escape", "esc":
		return "\u001b"
	case "backspace":
		return "\b"
	case "space":
		return " "
	default:
		return key
	}
}
