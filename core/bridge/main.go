package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	nativeDir, err := locateNativeDir()
	if err != nil {
		fatal(err)
	}

	cmd := exec.Command("cargo", "run", "--quiet")
	cmd.Dir = nativeDir
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fatal(err)
	}

	if err := cmd.Start(); err != nil {
		fatal(err)
	}

	request := Request{
		Action:  "PING",
		Params:  map[string]any{},
		TraceID: "bridge-ping-1",
	}

	if err := json.NewEncoder(stdin).Encode(request); err != nil {
		_ = cmd.Wait()
		fatal(err)
	}
	if err := stdin.Close(); err != nil {
		_ = cmd.Wait()
		fatal(err)
	}

	var response Response
	if err := json.NewDecoder(stdout).Decode(&response); err != nil {
		_ = cmd.Wait()
		fatal(err)
	}

	if err := cmd.Wait(); err != nil {
		fatal(err)
	}

	if response.Status != "success" {
		fatal(fmt.Errorf("native error: %s", response.Error))
	}

	fmt.Println(response.Payload["message"])
}

func locateNativeDir() (string, error) {
	candidates := []string{"../../drivers/native", "drivers/native"}
	for _, candidate := range candidates {
		path := filepath.Clean(candidate)
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}
	}
	return "", errors.New("drivers/native not found")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
