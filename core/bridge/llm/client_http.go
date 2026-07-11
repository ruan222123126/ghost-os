package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	sseScannerInitialBuffer = 64 * 1024
	sseScannerMaxBuffer     = 10 * 1024 * 1024
)

type jsonRequestOptions struct {
	Path    string
	Body    any
	Headers map[string]string
	Stream  bool
}

func (c *Client) postJSON(ctx context.Context, path string, requestBody any, headers map[string]string) ([]byte, int, error) {
	req, err := c.newJSONRequest(ctx, jsonRequestOptions{
		Path:    path,
		Body:    requestBody,
		Headers: headers,
	})
	if err != nil {
		return nil, 0, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("read response body: %w", err)
	}
	return raw, resp.StatusCode, nil
}

func (c *Client) streamJSON(
	ctx context.Context,
	path string,
	requestBody any,
	headers map[string]string,
	lineHandler func([]byte) error,
) error {
	req, err := c.newJSONRequest(ctx, jsonRequestOptions{
		Path:    path,
		Body:    requestBody,
		Headers: headers,
		Stream:  true,
	})
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if err := ensureStreamResponseStatus(resp); err != nil {
		return err
	}
	return scanSSEPayload(ctx, resp.Body, lineHandler)
}

func (c *Client) newJSONRequest(ctx context.Context, opts jsonRequestOptions) (*http.Request, error) {
	reqBody, err := json.Marshal(opts.Body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.opts.BaseURL + opts.Path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if opts.Stream {
		req.Header.Set("Accept", "text/event-stream")
	}
	applyHeaders(req.Header, opts.Headers)
	return req, nil
}

func ensureStreamResponseStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return fmt.Errorf("read response body: %w", readErr)
	}
	return ensureSuccessStatus(resp.StatusCode, raw)
}

var errSSEStreamDone = errors.New("sse stream done")

type sseDataCollector struct {
	lineHandler func([]byte) error
	dataLines   []string
}

func scanSSEPayload(ctx context.Context, body io.Reader, lineHandler func([]byte) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, sseScannerInitialBuffer), sseScannerMaxBuffer)

	collector := sseDataCollector{
		lineHandler: lineHandler,
		dataLines:   make([]string, 0, 1),
	}
	for scanner.Scan() {
		if err := collector.onLine(scanner.Text()); err != nil {
			if errors.Is(err, errSSEStreamDone) {
				return nil
			}
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return fmt.Errorf("stream interrupted: %w", err)
	}
	if err := collector.flush(); err != nil && !errors.Is(err, errSSEStreamDone) {
		return err
	}
	return nil
}

func (c *sseDataCollector) onLine(raw string) error {
	line := strings.TrimRight(raw, "\r")
	switch {
	case line == "":
		return c.flush()
	case strings.HasPrefix(line, ":"):
		return nil
	case strings.HasPrefix(line, "data:"):
		c.dataLines = append(c.dataLines, parseSSEDataLine(line))
	}
	return nil
}

func parseSSEDataLine(line string) string {
	value := strings.TrimPrefix(line, "data:")
	return strings.TrimPrefix(value, " ")
}

func (c *sseDataCollector) flush() error {
	if len(c.dataLines) == 0 {
		return nil
	}

	payload := strings.Join(c.dataLines, "\n")
	c.dataLines = c.dataLines[:0]
	if strings.TrimSpace(payload) == "[DONE]" {
		return errSSEStreamDone
	}
	return c.lineHandler([]byte(payload))
}
