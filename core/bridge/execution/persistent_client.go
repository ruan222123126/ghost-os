package execution

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultPersistentHandshakeTimeout = time.Second
	persistentHandshakeTimeoutEnv     = "GHOST_NATIVE_PERSISTENT_HANDSHAKE_TIMEOUT"
)

// PersistentNativeClient 通过持久 native 子进程复用 framed RPC 通道。
type PersistentNativeClient struct {
	mu                sync.Mutex
	cmd               *exec.Cmd
	stdin             io.WriteCloser
	stdout            io.ReadCloser
	stdoutReader      *bufio.Reader
	waitCh            chan error
	binaryPath        string
	locator           nativeBinaryLocator
	nextReqID         uint64
	closed            bool
	fallback          bool
	verified          bool
	handshakeTimeout  time.Duration
	commandFactory    func(binaryPath string) *exec.Cmd
	allowedReadPaths  []string
	allowedWritePaths []string
	workingDir        string
}

func NewPersistentNativeClient() *PersistentNativeClient {
	return newPersistentNativeClientWithLocator(nativeBinaryLocator{})
}

func newPersistentNativeClientWithLocator(locator nativeBinaryLocator) *PersistentNativeClient {
	return &PersistentNativeClient{
		handshakeTimeout: persistentHandshakeTimeoutFromEnv(),
		locator:          locator,
	}
}

func (c *PersistentNativeClient) Call(
	ctx context.Context,
	action string,
	params map[string]any,
	traceID string,
) (map[string]any, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, errors.New("persistent native client is closed")
	}
	if c.fallback {
		return c.callOneShotLocked(ctx, newRequest(action, params, traceID))
	}
	if err := c.ensureStartedLocked(); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
		if !c.verified {
			if err := c.verifyPersistentProtocolLocked(ctx); err != nil {
				if errors.Is(err, errPersistentProtocolUnsupported) {
					c.fallback = true
					return c.callOneShotLocked(ctx, newRequest(action, params, traceID))
				}
				if errors.Is(err, context.DeadlineExceeded) {
					c.fallback = true
					return c.callOneShotLocked(ctx, newRequest(action, params, traceID))
				}
				return nil, err
			}
		if c.cmd == nil {
			if err := c.ensureStartedLocked(); err != nil {
				return nil, err
			}
		}
	}

	req := newRequest(action, params, traceID)
	req.RequestID = c.nextRequestIDLocked()

	if err := writeFrame(c.stdin, req); err != nil {
		_ = c.stopProcessLocked(true)
		return nil, fmt.Errorf("write native frame failed: %w", err)
	}

	resp, err := c.readFrameWithContextLocked(ctx)
	if err != nil {
		if !c.verified && errors.Is(err, errPersistentProtocolUnsupported) {
			c.fallback = true
			return c.callOneShotLocked(ctx, newRequest(action, params, traceID))
		}
		return nil, err
	}
	defer c.reapExitedProcessLocked()
	if resp.RequestID != req.RequestID {
		_ = c.stopProcessLocked(true)
		return nil, fmt.Errorf(
			"unexpected native response request_id: got %q want %q",
			resp.RequestID,
			req.RequestID,
		)
	}

	c.verified = true
	payload, err := resp.intoResult(req)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (c *PersistentNativeClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true
	return c.stopProcessLocked(true)
}

func (c *PersistentNativeClient) SetWorkingDir(dir string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("persistent native client is closed")
	}
	trimmed := strings.TrimSpace(dir)
	if c.workingDir == trimmed {
		return nil
	}
	c.workingDir = trimmed
	c.verified = false
	return c.stopProcessLocked(true)
}

func (c *PersistentNativeClient) ensureStartedLocked() error {
	if c.closed {
		return errors.New("persistent native client is closed")
	}
	if c.cmd != nil {
		select {
		case <-c.waitCh:
			c.clearProcessLocked()
		default:
			return nil
		}
	}

	nativeBin := c.binaryPath
	if nativeBin == "" && c.commandFactory == nil {
		resolved, err := locateNativeBinary(c.locator)
		if err != nil {
			return err
		}
		nativeBin = resolved
		c.binaryPath = resolved
	}

	cmd := c.newCommand(nativeBin)
	c.applyWorkingDir(cmd)
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), cmd.Env...)
	cmd.Env = append(cmd.Env, buildNativeAllowedPathEnv(c.allowedReadPaths, c.allowedWritePaths)...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return err
	}

	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	c.cmd = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stdoutReader = bufio.NewReader(stdout)
	c.waitCh = waitCh
	return nil
}

func (c *PersistentNativeClient) newCommand(binaryPath string) *exec.Cmd {
	if c.commandFactory != nil {
		return c.commandFactory(binaryPath)
	}
	return exec.Command(binaryPath, "--persistent")
}

func (c *PersistentNativeClient) nextRequestIDLocked() string {
	c.nextReqID++
	return "native-req-" + strconv.FormatUint(c.nextReqID, 10)
}

func (c *PersistentNativeClient) readFrameWithContextLocked(ctx context.Context) (*response, error) {
	type result struct {
		response response
		err      error
	}

	resultCh := make(chan result, 1)
	go func(reader *bufio.Reader) {
		var resp response
		err := readFrame(reader, &resp)
		resultCh <- result{response: resp, err: err}
	}(c.stdoutReader)

	select {
	case <-ctx.Done():
		stopErr := c.stopProcessLocked(true)
		if stopErr != nil {
			return nil, fmt.Errorf("%w (native shutdown: %v)", ctx.Err(), stopErr)
		}
		return nil, ctx.Err()
	case res := <-resultCh:
		if res.err != nil {
			_ = c.stopProcessLocked(true)
			return nil, fmt.Errorf("read native frame failed: %w", res.err)
		}
		return &res.response, nil
	}
}

func (c *PersistentNativeClient) stopProcessLocked(kill bool) error {
	var stopErr error

	if kill && c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
			stopErr = err
		}
	}
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	if c.waitCh != nil {
		if err := <-c.waitCh; err != nil && !errors.Is(err, os.ErrProcessDone) {
			if stopErr == nil {
				stopErr = err
			}
		}
	}

	c.clearProcessLocked()
	return stopErr
}

func (c *PersistentNativeClient) clearProcessLocked() {
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.stdout != nil {
		_ = c.stdout.Close()
	}
	c.cmd = nil
	c.stdin = nil
	c.stdout = nil
	c.stdoutReader = nil
	c.waitCh = nil
}

func (c *PersistentNativeClient) reapExitedProcessLocked() {
	if c.waitCh == nil {
		return
	}
	select {
	case <-c.waitCh:
		c.clearProcessLocked()
	default:
	}
}

func (c *PersistentNativeClient) callOneShotLocked(ctx context.Context, req request) (map[string]any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c.commandFactory != nil {
		cmd := c.commandFactory(c.binaryPath)
		c.applyWorkingDir(cmd)
		return callNativeOnceWithCommand(
			cmd,
			req,
			c.allowedReadPaths,
			c.allowedWritePaths,
		)
	}
	if c.binaryPath == "" {
		resolved, err := locateNativeBinary(c.locator)
		if err != nil {
			return nil, err
		}
		c.binaryPath = resolved
	}
	cmd := exec.CommandContext(ctx, c.binaryPath)
	c.applyWorkingDir(cmd)
	return callNativeOnceWithCommand(
		cmd,
		req,
		c.allowedReadPaths,
		c.allowedWritePaths,
	)
}

func (c *PersistentNativeClient) applyWorkingDir(cmd *exec.Cmd) {
	if c == nil || cmd == nil {
		return
	}
	if dir := strings.TrimSpace(c.workingDir); dir != "" {
		cmd.Dir = dir
	}
}

func (c *PersistentNativeClient) verifyPersistentProtocolLocked(ctx context.Context) error {
	timeout := c.handshakeTimeout
	if timeout <= 0 {
		timeout = defaultPersistentHandshakeTimeout
	}
	handshakeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req := newRequest("PING", map[string]any{}, "bridge-exec-persistent-handshake")
	req.RequestID = c.nextRequestIDLocked()
	if err := writeFrame(c.stdin, req); err != nil {
		_ = c.stopProcessLocked(true)
		return fmt.Errorf("write persistent handshake failed: %w", err)
	}

	resp, err := c.readFrameWithContextLocked(handshakeCtx)
	if err != nil {
		return err
	}
	if resp.RequestID != req.RequestID {
		_ = c.stopProcessLocked(true)
		return fmt.Errorf(
			"unexpected native handshake request_id: got %q want %q",
			resp.RequestID,
			req.RequestID,
		)
	}
	if _, err := resp.intoResult(req); err != nil {
		return fmt.Errorf("persistent handshake failed: %w", err)
	}

	c.verified = true
	c.reapExitedProcessLocked()
	return nil
}

func persistentHandshakeTimeoutFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv(persistentHandshakeTimeoutEnv))
	if raw == "" {
		return defaultPersistentHandshakeTimeout
	}
	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}
	if ms, err := strconv.Atoi(raw); err == nil && ms > 0 {
		return time.Duration(ms) * time.Millisecond
	}
	return defaultPersistentHandshakeTimeout
}
