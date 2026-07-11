package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

const (
	langsmithFeedbackKey = "user_score"
	langsmithTimeout     = 10 * time.Second
)

// feedbackMirror pushes a thumb rating to an external trace store, best-effort.
type feedbackMirror interface {
	enabled() bool
	upsertFeedback(feedbackID, runID string, score float64, comment string)
}

// langsmithClient mirrors feedback to LangSmith's REST feedback endpoint.
type langsmithClient struct {
	endpoint string
	apiKey   string
	http     *http.Client
}

func newLangsmithClient(endpoint, apiKey string) *langsmithClient {
	return &langsmithClient{endpoint: endpoint, apiKey: apiKey, http: &http.Client{Timeout: langsmithTimeout}}
}

func (c *langsmithClient) enabled() bool { return c != nil && c.apiKey != "" }

// upsertFeedback mirrors a rating without blocking; errors are logged only.
func (c *langsmithClient) upsertFeedback(feedbackID, runID string, score float64, comment string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), langsmithTimeout)
		defer cancel()
		if err := c.post(ctx, feedbackID, runID, score, comment); err != nil {
			slog.Warn("langsmith feedback mirror failed", "err", err)
		}
	}()
}

// post sends one idempotent feedback upsert; returns an error for logging.
func (c *langsmithClient) post(ctx context.Context, feedbackID, runID string, score float64, comment string) error {
	body := map[string]any{"id": feedbackID, "run_id": runID, "key": langsmithFeedbackKey, "score": score}
	if comment != "" {
		body["comment"] = comment
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/feedback", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("langsmith: status %d", resp.StatusCode)
	}
	return nil
}

// newUUID returns a random RFC-4122 v4 UUID string.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
