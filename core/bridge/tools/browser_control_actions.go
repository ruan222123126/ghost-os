package tools

import (
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"

	"github.com/chromedp/chromedp"
)

func (t *BrowserControlTool) executeGoto(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	url, err := toolparams.RequiredString(params, "url")
	if err != nil {
		return "", err
	}

	var title string
	var location string
	actions, err := browserGotoActions(url, params, &title, &location)
	if err != nil {
		return "", err
	}
	if err := session.run(browserParseTimeout(params), actions...); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "goto",
		"session_id": session.id,
		"url":        location,
		"title":      title,
	})
}

func (t *BrowserControlTool) executeClick(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(toolparams.OptionalString(params, "selector_type", "css")) {
	case "text":
		return executeTextClick(session, params)
	case "xpath":
		return executeXPathClick(session, params)
	case "", "css":
		return executeCSSClick(session, params)
	default:
		return "", fmt.Errorf("selector_type must be one of: css, xpath, text")
	}
}

func (t *BrowserControlTool) executeType(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	selector, err := toolparams.RequiredString(params, "selector")
	if err != nil {
		return "", err
	}
	text, err := toolparams.RequiredString(params, "text")
	if err != nil {
		return "", err
	}

	actions := make([]chromedp.Action, 0, 3)
	if toolparams.OptionalBool(params, "clear", false) {
		actions = append(actions, chromedp.SetValue(selector, "", chromedp.ByQuery))
	}
	actions = append(actions, chromedp.Focus(selector, chromedp.ByQuery))
	actions = append(actions, chromedp.SendKeys(selector, text, chromedp.ByQuery))
	if err := session.run(browserParseTimeout(params), actions...); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "type",
		"session_id": session.id,
		"selector":   selector,
		"length":     len(text),
	})
}

func (t *BrowserControlTool) executePress(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	key, err := toolparams.RequiredString(params, "key")
	if err != nil {
		return "", err
	}
	selector := toolparams.OptionalString(params, "selector", "body")
	if err := session.run(browserParseTimeout(params), chromedp.SendKeys(selector, normalizeKeyPress(key), chromedp.ByQuery)); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "press",
		"session_id": session.id,
		"key":        key,
		"selector":   selector,
	})
}

func (t *BrowserControlTool) executeEvaluate(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	expr, err := toolparams.RequiredString(params, "expression")
	if err != nil {
		return "", err
	}
	var value any
	if err := session.run(browserParseTimeout(params), chromedp.Evaluate(expr, &value, chromedp.EvalAsValue)); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "evaluate",
		"session_id": session.id,
		"value":      value,
	})
}

func (t *BrowserControlTool) executeContent(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	selector := toolparams.OptionalString(params, "selector", "html")
	var html string
	if err := session.run(browserParseTimeout(params), chromedp.OuterHTML(selector, &html, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "content",
		"session_id": session.id,
		"selector":   selector,
		"content":    html,
	})
}

func (t *BrowserControlTool) executeInfo(params map[string]any) (string, error) {
	session, err := t.sessionFromParams(params)
	if err != nil {
		return "", err
	}
	var title string
	var location string
	if err := session.run(browserParseTimeout(params), chromedp.Location(&location), chromedp.Title(&title)); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":     "info",
		"session_id": session.id,
		"url":        location,
		"title":      title,
	})
}

func (t *BrowserControlTool) executeClose(params map[string]any) (string, error) {
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		session, err := t.singleSession()
		if err != nil {
			return "", err
		}
		sessionID = session.id
	}
	session := t.popSession(sessionID)
	if session == nil {
		return "", fmt.Errorf("browser session %q not found", sessionID)
	}
	session.close()
	return tooljson.Encode(map[string]any{
		"action":     "close",
		"session_id": sessionID,
		"closed":     true,
	})
}

func browserGotoActions(
	url string,
	params map[string]any,
	title *string,
	location *string,
) ([]chromedp.Action, error) {
	waitMode := strings.ToLower(toolparams.OptionalString(params, "wait", "dom"))
	var readyState string
	actions := []chromedp.Action{chromedp.Navigate(url)}
	switch waitMode {
	case "none":
	case "dom":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
	case "load":
		actions = append(actions, chromedp.WaitReady("body", chromedp.ByQuery))
		actions = append(actions, chromedp.Evaluate(`document.readyState`, &readyState))
	default:
		return nil, fmt.Errorf("wait must be one of: none, dom, load")
	}
	if extraWait := browserParseWaitDuration(params, "wait_ms"); extraWait > 0 {
		actions = append(actions, chromedp.Sleep(extraWait))
	}
	actions = append(actions, chromedp.Location(location), chromedp.Title(title))
	return actions, nil
}

func executeTextClick(session *browserSession, params map[string]any) (string, error) {
	text, err := toolparams.RequiredString(params, "text")
	if err != nil {
		return "", err
	}
	var clicked bool
	if err := session.run(browserParseTimeout(params), chromedp.Evaluate(clickByTextScript(text), &clicked)); err != nil {
		return "", err
	}
	if !clicked {
		return "", fmt.Errorf("no element matched text %q", text)
	}
	return tooljson.Encode(map[string]any{
		"action":     "click",
		"session_id": session.id,
		"selector":   "text",
		"text":       text,
	})
}

func executeXPathClick(session *browserSession, params map[string]any) (string, error) {
	selector, err := toolparams.RequiredString(params, "selector")
	if err != nil {
		return "", err
	}
	var clicked bool
	if err := session.run(browserParseTimeout(params), chromedp.Evaluate(clickByXPathScript(selector), &clicked)); err != nil {
		return "", err
	}
	if !clicked {
		return "", fmt.Errorf("no element matched xpath %q", selector)
	}
	return tooljson.Encode(map[string]any{
		"action":        "click",
		"session_id":    session.id,
		"selector":      selector,
		"selector_type": "xpath",
	})
}

func executeCSSClick(session *browserSession, params map[string]any) (string, error) {
	selector, err := toolparams.RequiredString(params, "selector")
	if err != nil {
		return "", err
	}
	if err := session.run(browserParseTimeout(params), chromedp.Click(selector, chromedp.ByQuery)); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":        "click",
		"session_id":    session.id,
		"selector":      selector,
		"selector_type": "css",
	})
}
