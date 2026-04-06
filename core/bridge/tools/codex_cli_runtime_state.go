package tools

import (
	"bufio"
	"io"
	"os/exec"
	"sync"
)

type codexCLICommandManager struct {
	mu       sync.Mutex
	nextSeq  uint64
	commands map[string]*codexCLICommand
}

type codexCLICommand struct {
	seq        uint64
	id         string
	outputPath string
	process    *exec.Cmd
	mu         sync.Mutex
	output     string
	sessionID  string
	exitCode   *int
}

type codexCLICommandSnapshot struct {
	sessionID  string
	exitCode   *int
	outputTail string
	outputPath string
}

func newCodexCLICommandManager() *codexCLICommandManager {
	return &codexCLICommandManager{commands: make(map[string]*codexCLICommand)}
}

func (m *codexCLICommandManager) nextCommandID() (uint64, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextSeq++
	return m.nextSeq, "codex-cli-" + formatUint(m.nextSeq)
}

func (m *codexCLICommandManager) insert(command *codexCLICommand) {
	if m == nil || command == nil {
		return
	}
	m.mu.Lock()
	m.commands[command.id] = command
	m.mu.Unlock()
}

func (m *codexCLICommandManager) find(commandID string, sessionID string) *codexCLICommand {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if command := m.findByCommandID(commandID); command != nil {
		return command
	}
	if command := m.findByCommandID(sessionID); command != nil {
		return command
	}
	return m.findLatestBySessionID(sessionID)
}

func (m *codexCLICommandManager) findByCommandID(commandID string) *codexCLICommand {
	if commandID == "" {
		return nil
	}
	command, ok := m.commands[commandID]
	if !ok {
		return nil
	}
	return command
}

func (m *codexCLICommandManager) findLatestBySessionID(sessionID string) *codexCLICommand {
	if sessionID == "" {
		return nil
	}
	var best *codexCLICommand
	for _, command := range m.commands {
		if command.sessionIDSnapshot() != sessionID {
			continue
		}
		if best == nil || command.seq > best.seq {
			best = command
		}
	}
	return best
}

func newCodexCLICommand(
	seq uint64,
	id string,
	outputPath string,
	process *exec.Cmd,
) *codexCLICommand {
	return &codexCLICommand{
		seq:        seq,
		id:         id,
		outputPath: outputPath,
		process:    process,
	}
}

func (c *codexCLICommand) startOutputReaders(stdout io.ReadCloser, stderr io.ReadCloser) {
	if stdout != nil {
		go c.consumeOutput(stdout, "STDOUT: ")
	}
	if stderr != nil {
		go c.consumeOutput(stderr, "STDERR: ")
	}
}

func (c *codexCLICommand) consumeOutput(reader io.ReadCloser, prefix string) {
	defer reader.Close()
	buffer := bufio.NewReader(reader)
	for {
		line, err := buffer.ReadString('\n')
		if len(line) > 0 {
			c.recordOutputLine(line, prefix)
		}
		if err != nil {
			return
		}
	}
}

func (c *codexCLICommand) recordOutputLine(line string, prefix string) {
	if c == nil || line == "" {
		return
	}
	if sessionID := parseCodexCLISessionID(line); sessionID != "" {
		c.setSessionID(sessionID)
	}
	c.appendOutput(prefix + line)
}

func (c *codexCLICommand) appendOutput(text string) {
	if text == "" {
		return
	}
	c.mu.Lock()
	c.output += text
	c.output = trimToLastChars(c.output, codexCLIMaxOutputBufferChars)
	c.mu.Unlock()
}

func (c *codexCLICommand) setSessionID(sessionID string) {
	if sessionID == "" {
		return
	}
	c.mu.Lock()
	if c.sessionID == "" {
		c.sessionID = sessionID
	}
	c.mu.Unlock()
}

func (c *codexCLICommand) sessionIDSnapshot() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

func (c *codexCLICommand) outputPathSnapshot() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.outputPath
}

func (c *codexCLICommand) startWaiter() {
	if c == nil || c.process == nil {
		return
	}
	go c.waitForExit()
}

func (c *codexCLICommand) waitForExit() {
	waitErr := c.process.Wait()
	code := codexCLIUnknownExitCode
	if state := c.process.ProcessState; state != nil {
		exitCode := state.ExitCode()
		if exitCode != -1 {
			code = exitCode
		}
	}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		if exitCode := exitErr.ExitCode(); exitCode != -1 {
			code = exitCode
		}
	}
	c.setExitCode(code)
}

func (c *codexCLICommand) setExitCode(exitCode int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.exitCode != nil {
		return
	}
	value := exitCode
	c.exitCode = &value
}

func (c *codexCLICommand) snapshot(outputChars int) codexCLICommandSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := codexCLICommandSnapshot{
		sessionID:  c.sessionID,
		outputTail: trimToLastChars(c.output, outputChars),
		outputPath: c.outputPath,
	}
	if c.exitCode != nil {
		value := *c.exitCode
		result.exitCode = &value
	}
	return result
}
