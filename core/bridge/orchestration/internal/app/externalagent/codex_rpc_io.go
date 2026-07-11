package externalagent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync/atomic"
)

func (c *appServerClient) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(ctx, rpcRequestTimeout)
	defer cancel()

	id := int(atomic.AddInt64(&c.nextID, 1))
	ch := make(chan rpcResponse, 1)
	if err := c.writeRequest(id, method, params, ch); err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.removePending(id)
		return nil, fmt.Errorf("%s timed out or cancelled: %w", method, ctx.Err())
	case response, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("%s failed: codex process disconnected", method)
		}
		if response.Error != nil {
			return nil, fmt.Errorf("%s: %s (code=%d)", method, response.Error.Message, response.Error.Code)
		}
		return response.Result, nil
	}
}

func (c *appServerClient) writeRequest(id int, method string, params any, ch chan rpcResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return fmt.Errorf("cannot send %s: codex stdin is not available", method)
	}
	c.pending[id] = ch
	return c.writeLocked(rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params})
}

func (c *appServerClient) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return fmt.Errorf("cannot send %s: codex stdin is not available", method)
	}
	return c.writeLocked(rpcRequest{JSONRPC: "2.0", Method: method, Params: params})
}

func (c *appServerClient) respond(id int, result any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return
	}
	msg := map[string]any{"jsonrpc": "2.0", "id": id, "result": result}
	if err := c.writeLocked(msg); err != nil {
		log.Printf("codex app-server response failed id=%d error=%v", id, err)
	}
}

func (c *appServerClient) writeLocked(msg any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = c.stdin.Write(data)
	return err
}

func (c *appServerClient) removePending(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *appServerClient) readStdout(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		c.handleLine(scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		log.Printf("codex app-server stdout error: %v", err)
	}
}

func (c *appServerClient) readStderr(stderr io.Reader) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			log.Printf("codex app-server stderr: %s", line)
		}
	}
}

func (c *appServerClient) waitProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		log.Printf("codex app-server exited: %v", err)
	}
	c.mu.Lock()
	if c.cmd == cmd {
		c.cmd = nil
		c.stdin = nil
		c.connected = false
	}
	pending := c.pending
	c.pending = make(map[int]chan rpcResponse)
	c.mu.Unlock()
	for _, ch := range pending {
		close(ch)
	}
}

func (c *appServerClient) handleLine(line []byte) {
	var envelope rpcEnvelope
	if err := json.Unmarshal(line, &envelope); err != nil {
		log.Printf("codex app-server non-json line: %s", strings.TrimSpace(string(line)))
		return
	}
	if envelope.ID != nil && (envelope.Result != nil || envelope.Error != nil) && envelope.Method == "" {
		c.handleResponse(*envelope.ID, envelope)
		return
	}
	if envelope.ID != nil && envelope.Method != "" {
		go c.handleServerRequest(*envelope.ID, envelope.Method, envelope.Params)
		return
	}
	if envelope.Method != "" {
		c.handleNotification(envelope.Method, envelope.Params)
	}
}

func (c *appServerClient) handleResponse(id int, envelope rpcEnvelope) {
	c.mu.Lock()
	ch := c.pending[id]
	delete(c.pending, id)
	c.mu.Unlock()
	if ch == nil {
		return
	}
	ch <- rpcResponse{ID: id, Result: envelope.Result, Error: envelope.Error}
}

func (c *appServerClient) handleServerRequest(id int, method string, raw json.RawMessage) {
	request, handled := approvalRequestFromRPC(id, method, raw)
	if !handled {
		c.respond(id, map[string]any{})
		return
	}
	handler := c.approvalHandlerSnapshot()
	decision := DecisionDenied
	if handler != nil {
		resolved, err := handler(context.Background(), request)
		if err != nil {
			log.Printf("codex approval handler failed method=%s id=%d error=%v", method, id, err)
		} else if normalized, err := ValidateApprovalDecision(resolved); err == nil {
			decision = normalized
		}
	}
	if request.MCP {
		c.respond(id, decisionToMCPResponse(decision))
		return
	}
	c.respond(id, map[string]any{"decision": decisionToWire(decision, request.Legacy)})
}

func (c *appServerClient) approvalHandlerSnapshot() func(context.Context, ApprovalRequest) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.approvalHandler
}

func (c *appServerClient) handleNotification(method string, raw json.RawMessage) {
	event, ok := eventFromNotification(method, raw)
	if !ok {
		return
	}
	handler := c.eventHandlerSnapshot()
	if handler != nil {
		handler(event)
	}
}

func (c *appServerClient) eventHandlerSnapshot() func(CodexEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.eventHandler
}
