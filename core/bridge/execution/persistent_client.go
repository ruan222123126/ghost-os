package execution

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os/exec"
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
	return c.callLocked(ctx, action, params, traceID)
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
