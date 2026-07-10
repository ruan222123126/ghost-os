package externalagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func (m *Manager) ListModels(ctx context.Context) (ModelCatalog, error) {
	if m == nil || m.ConfigStore == nil {
		return ModelCatalog{}, errors.New("external agent manager is not configured")
	}
	cfg, err := m.ConfigStore.Config()
	if err != nil {
		return ModelCatalog{}, err
	}
	factory := m.ClientFactory
	if factory == nil {
		factory = DefaultClientFactory
	}
	client := factory(ClientConfig{
		CodexPath: cfg.CodexCLIPath,
		NodePath:  cfg.NodeBinPath,
		CWD:       defaultString(strings.TrimSpace(cfg.ProjectRoot), "."),
	})
	if err := client.Connect(ctx); err != nil {
		return ModelCatalog{}, err
	}
	defer client.Close()
	return client.ListModels(ctx)
}

type modelPage struct {
	Data []struct {
		Model     string `json:"model"`
		IsDefault bool   `json:"isDefault"`
	} `json:"data"`
	NextCursor string `json:"nextCursor"`
}

func (c *appServerClient) ListModels(ctx context.Context) (ModelCatalog, error) {
	cursor := ""
	models := make([]string, 0)
	seenModels := make(map[string]struct{})
	seenCursors := make(map[string]struct{})
	defaultModel := ""
	for {
		page, err := c.loadModelPage(ctx, cursor)
		if err != nil {
			return ModelCatalog{}, err
		}
		models, defaultModel = collectModelPage(models, seenModels, page, defaultModel)
		nextCursor := strings.TrimSpace(page.NextCursor)
		if nextCursor == "" {
			break
		}
		if _, exists := seenCursors[nextCursor]; exists {
			return ModelCatalog{}, fmt.Errorf("model/list returned repeated cursor %q", nextCursor)
		}
		seenCursors[nextCursor] = struct{}{}
		cursor = nextCursor
	}
	if len(models) == 0 {
		return ModelCatalog{}, fmt.Errorf("model/list returned no visible models")
	}
	if defaultModel == "" {
		return ModelCatalog{}, fmt.Errorf("model/list returned no default model")
	}
	return ModelCatalog{Models: models, DefaultModel: defaultModel}, nil
}

func (c *appServerClient) loadModelPage(ctx context.Context, cursor string) (modelPage, error) {
	params := map[string]any{"includeHidden": false}
	if cursor != "" {
		params["cursor"] = cursor
	}
	raw, err := c.request(ctx, "model/list", params)
	if err != nil {
		return modelPage{}, err
	}
	var page modelPage
	if err := json.Unmarshal(raw, &page); err != nil {
		return modelPage{}, fmt.Errorf("decode model/list response: %w", err)
	}
	return page, nil
}

func collectModelPage(models []string, seen map[string]struct{}, page modelPage, defaultModel string) ([]string, string) {
	for _, item := range page.Data {
		model := strings.TrimSpace(item.Model)
		if model == "" {
			continue
		}
		if _, exists := seen[model]; !exists {
			seen[model] = struct{}{}
			models = append(models, model)
		}
		if item.IsDefault {
			defaultModel = model
		}
	}
	return models, defaultModel
}

func sandboxPolicy(sandbox string) map[string]any {
	switch sandbox {
	case "read-only":
		return map[string]any{"type": "readOnly"}
	case "danger-full-access":
		return map[string]any{"type": "dangerFullAccess"}
	default:
		return map[string]any{"type": "workspaceWrite"}
	}
}

func (c *appServerClient) InterruptTurn(ctx context.Context, threadID string, turnID string) error {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
		return nil
	}
	_, err := c.request(ctx, "turn/interrupt", map[string]string{
		"threadId": strings.TrimSpace(threadID),
		"turnId":   strings.TrimSpace(turnID),
	})
	return err
}

func (c *appServerClient) Close() error {
	c.mu.Lock()
	cmd := c.cmd
	stdin := c.stdin
	c.cmd = nil
	c.stdin = nil
	c.connected = false
	pending := c.pending
	c.pending = make(map[int]chan rpcResponse)
	c.mu.Unlock()

	for _, ch := range pending {
		close(ch)
	}
	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}
