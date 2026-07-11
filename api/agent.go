package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// Upstream timeouts for the agent service.
const (
	agentDialTimeout           = 10 * time.Second
	agentTLSTimeout            = 10 * time.Second
	agentResponseHeaderTimeout = 30 * time.Second
)

// agentClient streams runs from the agent service's /run SSE endpoint.
type agentClient struct {
	baseURL string
	http    *http.Client
	token   func(context.Context) (string, error) // nil = no auth (local http)
}

// llmMessage is one conversation turn sent to the agent.
type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// nodeFrame is a node announced by the agent's event log.
type nodeFrame struct {
	ID       string          `json:"id"`
	ParentID *string         `json:"parent_id"`
	Type     string          `json:"type"`
	Name     string          `json:"name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

// runHandlers receives the agent's streamed node frames.
type runHandlers struct {
	onNode    func(nodeFrame)
	onDelta   func(id, text string)
	onNodeEnd func(id string, output json.RawMessage)
	onMeta    func(runID string)
}

// run POSTs history to /run, streams events, and returns aggregate usage.
// threadID groups the conversation's runs into one LangSmith thread.
func (c *agentClient) run(ctx context.Context, msgs []llmMessage, threadID string, h runHandlers) (tokenUsage, error) {
	var usage tokenUsage

	payload := map[string]any{"messages": msgs}
	if threadID != "" {
		payload["thread_id"] = threadID
	}
	reqBody, err := json.Marshal(payload)
	if err != nil {
		return usage, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/run", bytes.NewReader(reqBody))
	if err != nil {
		return usage, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != nil {
		tok, err := c.token(ctx)
		if err != nil {
			return usage, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return usage, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return usage, fmt.Errorf("agent: status %d", resp.StatusCode)
	}

	// SSE frames: "event: <type>\ndata: <json>\n\n".
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			event = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			if err := dispatchEvent(event, strings.TrimPrefix(line, "data: "), &usage, h); err != nil {
				return usage, err
			}
		}
	}
	return usage, scanner.Err()
}

// dispatchEvent handles one SSE frame; an "error" event returns a run error.
func dispatchEvent(event, data string, usage *tokenUsage, h runHandlers) error {
	switch event {
	case "node":
		var n nodeFrame
		if json.Unmarshal([]byte(data), &n) == nil && h.onNode != nil {
			h.onNode(n)
		}
	case "delta":
		var d struct {
			ID   string `json:"id"`
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(data), &d) == nil && h.onDelta != nil {
			h.onDelta(d.ID, d.Text)
		}
	case "node_end":
		var d struct {
			ID     string          `json:"id"`
			Output json.RawMessage `json:"output"`
		}
		if json.Unmarshal([]byte(data), &d) == nil && h.onNodeEnd != nil {
			h.onNodeEnd(d.ID, d.Output)
		}
	case "usage":
		var u struct {
			Input  int `json:"input"`
			Output int `json:"output"`
		}
		if json.Unmarshal([]byte(data), &u) == nil {
			*usage = tokenUsage{Prompt: u.Input, Completion: u.Output}
		}
	case "meta":
		var m struct {
			RunID string `json:"langsmith_run_id"`
		}
		if json.Unmarshal([]byte(data), &m) == nil && h.onMeta != nil {
			h.onMeta(m.RunID)
		}
	case "error":
		var e struct {
			Message string `json:"message"`
		}
		_ = json.Unmarshal([]byte(data), &e)
		return fmt.Errorf("agent: %s", e.Message)
	}
	return nil
}

func newAgentHTTPClient() *http.Client {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = (&net.Dialer{Timeout: agentDialTimeout}).DialContext
	tr.TLSHandshakeTimeout = agentTLSTimeout
	tr.ResponseHeaderTimeout = agentResponseHeaderTimeout
	return &http.Client{Transport: tr} // no Timeout: would cut SSE
}
