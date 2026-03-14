package execution

import (
	"bufio"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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

func (c *PersistentNativeClient) applyWorkingDir(cmd *exec.Cmd) {
	if c == nil || cmd == nil {
		return
	}
	if dir := strings.TrimSpace(c.workingDir); dir != "" {
		cmd.Dir = dir
	}
}
