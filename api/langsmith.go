package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	langsmithScoreKey   = "user_score"
	langsmithCommentKey = "user_feedback"
	langsmithTimeout    = 10 * time.Second
)

// feedbackMirror pushes a rating and optional note to an external trace store.
type feedbackMirror interface {
	enabled() bool
	upsertScore(feedbackID, runID string, score float64)
	upsertComment(feedbackID, runID, comment string)
	deleteFeedback(feedbackID string)
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

// upsertScore mirrors the numeric thumb rating as the user_score feedback.
func (c *langsmithClient) upsertScore(feedbackID, runID string, score float64) {
	c.fire(func(ctx context.Context) error {
		return c.post(ctx, feedbackID, runID, langsmithScoreKey, &score, nil)
	})
}

// upsertComment mirrors the note as the string-valued user_feedback feedback.
func (c *langsmithClient) upsertComment(feedbackID, runID, comment string) {
	c.fire(func(ctx context.Context) error {
		return c.post(ctx, feedbackID, runID, langsmithCommentKey, nil, &comment)
	})
}

// deleteFeedback removes a feedback by id (used when a note is cleared).
func (c *langsmithClient) deleteFeedback(feedbackID string) {
	c.fire(func(ctx context.Context) error { return c.del(ctx, feedbackID) })
}

// fire runs a best-effort call off the request path; errors are logged only.
func (c *langsmithClient) fire(fn func(context.Context) error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), langsmithTimeout)
		defer cancel()
		if err := fn(ctx); err != nil {
			slog.Warn("langsmith feedback mirror failed", "err", err)
		}
	}()
}

// post upserts one feedback (idempotent on feedbackID) with a score or value.
func (c *langsmithClient) post(ctx context.Context, feedbackID, runID, key string, score *float64, value *string) error {
	body := map[string]any{"id": feedbackID, "run_id": runID, "key": key}
	if score != nil {
		body["score"] = *score
	}
	if value != nil {
		body["value"] = *value
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/feedback", bytes.NewReader(buf))
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

// del removes a feedback by id; a 404 is treated as already-absent.
func (c *langsmithClient) del(ctx context.Context, feedbackID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.endpoint+"/api/v1/feedback/"+feedbackID, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", c.apiKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("langsmith: status %d", resp.StatusCode)
	}
	return nil
}

// commentFeedbackID derives the note's stable feedback id from the score's.
func commentFeedbackID(scoreID string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(scoreID+":"+langsmithCommentKey)).String()
}
