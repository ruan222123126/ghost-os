package tools

import (
	"strings"
	"sync"
)

type codexCLICommandManager struct {
	mu       sync.Mutex
	nextSeq  uint64
	commands map[string]*codexCLICommand
}

type codexCLICommand struct {
	seq          uint64
	id           string
	outputPath   string
	exitCodePath string
	mu           sync.Mutex
	sessionID    string
	exitCode     *int
}

type codexCLICommandSnapshot struct {
	sessionID    string
	exitCode     *int
	outputPath   string
	exitCodePath string
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
	exitCodePath string,
) *codexCLICommand {
	return &codexCLICommand{
		seq:          seq,
		id:           id,
		outputPath:   strings.TrimSpace(outputPath),
		exitCodePath: strings.TrimSpace(exitCodePath),
	}
}

func (c *codexCLICommand) applyStatus(outputTail string, exitCode *int) {
	c.setSessionID(parseCodexCLISessionIDFromOutput(outputTail))
	if exitCode != nil {
		c.setExitCode(*exitCode)
	}
}

func (c *codexCLICommand) setSessionID(sessionID string) {
	if c == nil || sessionID == "" {
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

func (c *codexCLICommand) setExitCode(exitCode int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	value := exitCode
	c.exitCode = &value
}

func (c *codexCLICommand) snapshot() codexCLICommandSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := codexCLICommandSnapshot{
		sessionID:    c.sessionID,
		outputPath:   c.outputPath,
		exitCodePath: c.exitCodePath,
	}
	if c.exitCode != nil {
		value := *c.exitCode
		result.exitCode = &value
	}
	return result
}
