package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type Request struct {
	Action  string         `json:"action"`
	Params  map[string]any `json:"params"`
	TraceID string         `json:"trace_id"`
}

type Response struct {
	Status  string         `json:"status"`
	Payload map[string]any `json:"payload"`
	Error   string         `json:"error"`
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		if err := runAgent(context.Background(), "Hello, what can you do?"); err != nil {
			fatal(err)
		}
		return
	}

	switch args[0] {
	case "ping":
		if err := runPing(); err != nil {
			fatal(err)
		}
	case "agent":
		message := "Hello, what can you do?"
		if len(args) > 1 {
			message = strings.Join(args[1:], " ")
		}
		if err := runAgent(context.Background(), message); err != nil {
			fatal(err)
		}
	default:
		// If command is unknown, treat all args as the user prompt and run agent mode.
		if err := runAgent(context.Background(), strings.Join(args, " ")); err != nil {
			fatal(err)
		}
	}
}

func runAgent(ctx context.Context, userMessage string) error {
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	client := llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              cfg.Model,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	})
	registry := tools.NewRegistry()
	registry.Register(tools.NewBashExecTool())
	registry.Register(tools.NewListFilesTool())

	systemPrompt := "You are Ghost-OS bridge agent. Use tools when needed and keep answers concise."
	a := agent.NewAgent(client, registry, systemPrompt, cfg.MaxTurns)

	reply, err := a.Run(ctx, userMessage)
	if err != nil {
		return err
	}

	fmt.Println(reply)
	return nil
}

func runPing() error {
	nativeBin, err := locateNativeBinary()
	if err != nil {
		return err
	}

	cmd := exec.Command(nativeBin)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	request := Request{
		Action:  "PING",
		Params:  map[string]any{},
		TraceID: "bridge-ping-1",
	}

	if err := json.NewEncoder(stdin).Encode(request); err != nil {
		_ = cmd.Wait()
		return err
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		return err
	}

	var response Response
	if err := json.NewDecoder(stdout).Decode(&response); err != nil {
		_ = cmd.Wait()
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	if response.Status != "success" {
		return fmt.Errorf("native error: %s", response.Error)
	}

	fmt.Println(response.Payload["message"])
	return nil
}

func locateNativeBinary() (string, error) {
	candidates := []string{
		"../../drivers/native/target/debug/native",
		"drivers/native/target/debug/native",
		"../../drivers/native/target/debug/native.exe",
		"drivers/native/target/debug/native.exe",
	}

	for _, candidate := range candidates {
		path := filepath.Clean(candidate)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path, nil
		}
	}

	return "", errors.New("native binary not found, run `cargo build` in drivers/native first")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
