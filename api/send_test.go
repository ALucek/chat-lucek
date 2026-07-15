package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// fakeAgent returns a client hitting a server that emits the given SSE frames.
func fakeAgent(t *testing.T, status int, frames ...string) *agentClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		for _, f := range frames {
			fmt.Fprint(w, f)
		}
	}))
	t.Cleanup(srv.Close)
	return &agentClient{baseURL: srv.URL, http: srv.Client()}
}

// nodeStartFrame builds a `node` SSE frame; empty parent means top-level.
func nodeStartFrame(id, parent, typ, name, input string) string {
	p := "null"
	if parent != "" {
		p = fmt.Sprintf("%q", parent)
	}
	extra := ""
	if name != "" {
		extra += fmt.Sprintf(`,"name":%q`, name)
	}
	if input != "" {
		extra += fmt.Sprintf(`,"input":%s`, input)
	}
	return fmt.Sprintf("event: node\ndata: {\"id\":%q,\"parent_id\":%s,\"type\":%q%s}\n\n", id, p, typ, extra)
}

// deltaFrame builds a `delta` SSE frame appending text to a node.
func deltaFrame(id, text string) string {
	return fmt.Sprintf("event: delta\ndata: {\"id\":%q,\"text\":%q}\n\n", id, text)
}

// nodeEndFrame builds a `node_end` SSE frame carrying a tool's output.
func nodeEndFrame(id, output string) string {
	return fmt.Sprintf("event: node_end\ndata: {\"id\":%q,\"output\":%s}\n\n", id, output)
}

// textFrames opens a top-level text node and streams one delta into it.
func textFrames(id, text string) string {
	return nodeStartFrame(id, "", "text", "", "") + deltaFrame(id, text)
}

// usageFrame builds the aggregate-usage SSE frame.
func usageFrame(input, output int) string {
	return fmt.Sprintf(
		"event: usage\ndata: {\"input\":%d,\"output\":%d,\"total\":%d,\"reasoning\":0}\n\n",
		input, output, input+output)
}

const endFrame = "event: end\ndata: {}\n\n"

func TestSend_StreamsAndPersists(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK,
		nodeStartFrame("a:text", "", "text", "", ""),
		deltaFrame("a:text", "Hello"), deltaFrame("a:text", " there"), endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("want X-Accel-Buffering: no, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: node") || !strings.Contains(body, "event: delta") {
		t.Fatalf("missing node/delta frames: %s", body)
	}
	if !strings.Contains(body, `"text":"Hello"`) {
		t.Fatalf("missing delta text: %s", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Fatalf("missing done event: %s", body)
	}

	// Two rows persisted: user then assistant, assistant content concatenated.
	rows, err := testPool.Query(context.Background(),
		`select role, content from messages where conversation_id=$1 order by id`, cid)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var got []llmMessage
	for rows.Next() {
		var m llmMessage
		rows.Scan(&m.Role, &m.Content)
		got = append(got, m)
	}
	if len(got) != 2 || got[0].Role != "user" || got[1].Role != "assistant" || got[1].Content != "Hello there" {
		t.Fatalf("unexpected messages: %+v", got)
	}
}

func TestSend_PersistsRunID(t *testing.T) {
	resetDB(t)
	meta := "event: meta\ndata: {\"langsmith_run_id\":\"run-123\"}\n\n"
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), meta, endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	var runID *string
	if err := testPool.QueryRow(context.Background(),
		`select langsmith_run_id from messages where conversation_id=$1 and role='assistant'`,
		cid).Scan(&runID); err != nil {
		t.Fatalf("query: %v", err)
	}
	if runID == nil || *runID != "run-123" {
		t.Fatalf("want run-123, got %v", runID)
	}
}

func TestSend_NoRunIDWhenAbsent(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	var runID *string
	if err := testPool.QueryRow(context.Background(),
		`select langsmith_run_id from messages where conversation_id=$1 and role='assistant'`,
		cid).Scan(&runID); err != nil {
		t.Fatalf("query: %v", err)
	}
	if runID != nil {
		t.Fatalf("want nil run id, got %q", *runID)
	}
}

func TestSend_BumpsUpdatedAt(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	first := createConversation(t, mux, ta)
	createConversation(t, mux, ta) // second, newer

	// Send to the OLDER conversation; it should jump to the top of the list.
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", first), ta,
		map[string]string{"content": "hi"})

	var list []struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(do(t, mux, http.MethodGet, "/api/conversations", ta, nil).Body.Bytes(), &list)
	if len(list) != 2 || list[0].ID != first {
		t.Fatalf("sent-to conversation should sort first, got %+v (want first=%d)", list, first)
	}
}

func TestSend_NotOwner(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	tb, _ := signup(t, mux, "b@x.com")
	cid := createConversation(t, mux, ta)
	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), tb,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", rec.Code)
	}
}

func TestSend_EmptyContent(t *testing.T) {
	resetDB(t)
	mux := newTestMux(fakeAgent(t, http.StatusOK))
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestSend_BadID(t *testing.T) {
	resetDB(t)
	mux := newTestMux(fakeAgent(t, http.StatusOK))
	ta, _ := signup(t, mux, "a@x.com")
	rec := do(t, mux, http.MethodPost, "/api/conversations/abc/messages", ta,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

func TestSend_UpstreamError(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusInternalServerError)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})
	if !strings.Contains(rec.Body.String(), "event: error") {
		t.Fatalf("want error event, got %s", rec.Body)
	}
	// User message persisted; assistant NOT (persist only complete replies).
	var roles []string
	rows, _ := testPool.Query(context.Background(),
		`select role from messages where conversation_id=$1 order by id`, cid)
	defer rows.Close()
	for rows.Next() {
		var role string
		rows.Scan(&role)
		roles = append(roles, role)
	}
	if len(roles) != 1 || roles[0] != "user" {
		t.Fatalf("want only [user], got %v", roles)
	}
}

func TestSend_RecordsUsage(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), usageFrame(4, 6), endFrame)
	mux := newTestMux(client)
	ta, uid := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}

	var total int
	err := testPool.QueryRow(context.Background(),
		`select coalesce(sum(prompt_tokens + completion_tokens), 0) from token_usage where user_id=$1`,
		uid).Scan(&total)
	if err != nil {
		t.Fatalf("usage query: %v", err)
	}
	if total != 10 {
		t.Fatalf("want recorded usage 10, got %d", total)
	}
}

func TestSend_OverRunBudget(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	mux := newTestMuxBudget(client, 1)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	// Seed one run mark to reach the budget of 1.
	seedMarks(t, "a@x.com", 1)

	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429 over budget, got %d", rec.Code)
	}
}

func TestSend_MessageTooLong(t *testing.T) {
	resetDB(t)
	mux := newTestMux(fakeAgent(t, http.StatusOK))
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)

	long := strings.Repeat("x", maxMessageChars+1)
	rec := do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": long})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", rec.Code)
	}
}

// assistantTrace returns the conversation's assistant-message trace.
func assistantTrace(t *testing.T, mux http.Handler, token string, cid int64) json.RawMessage {
	t.Helper()
	var msgs []struct {
		Role  string          `json:"role"`
		Trace json.RawMessage `json:"trace"`
	}
	body := do(t, mux, http.MethodGet, fmt.Sprintf("/api/conversations/%d/messages", cid), token, nil).Body.Bytes()
	if err := json.Unmarshal(body, &msgs); err != nil {
		t.Fatalf("decode messages: %v", err)
	}
	for _, m := range msgs {
		if m.Role == "assistant" {
			return m.Trace
		}
	}
	t.Fatal("no assistant message found")
	return nil
}

func TestSend_PersistsNodeTrace(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK,
		nodeStartFrame("r1:reasoning", "", "reasoning", "", ""), deltaFrame("r1:reasoning", "thinking"),
		nodeStartFrame("SA", "", "tool", "run_subagent", `{"task":"research"}`),
		nodeStartFrame("s1", "SA", "tool", "internet_search", `{"query":"q"}`),
		nodeEndFrame("s1", `{"results":[]}`),
		nodeStartFrame("m:text", "SA", "text", "", ""), deltaFrame("m:text", "sub summary"),
		nodeEndFrame("SA", `"done"`),
		nodeStartFrame("a:text", "", "text", "", ""), deltaFrame("a:text", "Answer"),
		endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	// content is the top-level answer text only.
	var content string
	if err := testPool.QueryRow(context.Background(),
		`select content from messages where conversation_id=$1 and role='assistant'`, cid).Scan(&content); err != nil {
		t.Fatalf("content query: %v", err)
	}
	if content != "Answer" {
		t.Fatalf("content: %q", content)
	}

	var trace messageTrace
	if err := json.Unmarshal(assistantTrace(t, mux, ta, cid), &trace); err != nil {
		t.Fatalf("decode trace: %v", err)
	}
	if trace.Version != 2 {
		t.Fatalf("version: %d", trace.Version)
	}
	// the search nests under the subagent
	var s1 *traceNode
	for i := range trace.Nodes {
		if trace.Nodes[i].ID == "s1" {
			s1 = &trace.Nodes[i]
		}
	}
	if s1 == nil || s1.ParentID == nil || *s1.ParentID != "SA" {
		t.Fatalf("s1 nesting: %+v", s1)
	}
	// the subagent emitted a nested text node
	var hasSubText bool
	for _, n := range trace.Nodes {
		if n.ID == "m:text" && n.ParentID != nil && *n.ParentID == "SA" && n.Text == "sub summary" {
			hasSubText = true
		}
	}
	if !hasSubText {
		t.Fatalf("missing subagent text node: %+v", trace.Nodes)
	}
}

func TestSend_NoTraceForPlainAnswer(t *testing.T) {
	resetDB(t)
	client := fakeAgent(t, http.StatusOK, textFrames("a", "hi"), endFrame)
	mux := newTestMux(client)
	ta, _ := signup(t, mux, "a@x.com")
	cid := createConversation(t, mux, ta)
	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	if raw := assistantTrace(t, mux, ta, cid); len(raw) != 0 {
		t.Fatalf("plain answer should have no trace, got %s", raw)
	}
}

func TestConversations_HasSummaryColumns(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	var uid int64
	testPool.QueryRow(ctx, `insert into users (email) values ('c@x.com') returning id`).Scan(&uid)
	var cid int64
	testPool.QueryRow(ctx, `insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid)
	var mid int64
	if err := testPool.QueryRow(ctx,
		`insert into messages (conversation_id, role, content) values ($1,'user','hi') returning id`, cid).Scan(&mid); err != nil {
		t.Fatalf("seed message: %v", err)
	}
	if _, err := testPool.Exec(ctx,
		`update conversations set summary = 'recap', summary_through_message_id = $1 where id = $2`, mid, cid); err != nil {
		t.Fatalf("update summary: %v", err)
	}
	var sum *string
	var wm *int64
	if err := testPool.QueryRow(ctx,
		`select summary, summary_through_message_id from conversations where id = $1`, cid).Scan(&sum, &wm); err != nil {
		t.Fatalf("select summary: %v", err)
	}
	if sum == nil || *sum != "recap" || wm == nil || *wm != mid {
		t.Fatalf("summary=%v watermark=%v", sum, wm)
	}
}

func TestSend_AssemblesSummaryPlusTail(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, textFrames("a", "ok"), endFrame)
	}))
	t.Cleanup(srv.Close)
	client := &agentClient{baseURL: srv.URL, http: srv.Client()}
	mux := newTestMux(client)

	ta, uid := signup(t, mux, "a@x.com")
	var cid int64
	testPool.QueryRow(ctx, `insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid)
	var m2 int64
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'user','old-1') returning id`, cid).Scan(new(int64))
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'assistant','old-2') returning id`, cid).Scan(&m2)
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'user','old-3') returning id`, cid).Scan(new(int64))
	// Summary covers m2; only old-3 and the new turn are sent verbatim.
	testPool.Exec(ctx, `update conversations set summary='SUM', summary_through_message_id=$1 where id=$2`, m2, cid)

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "new-turn"})

	var sent struct {
		Messages []llmMessage `json:"messages"`
	}
	if err := json.Unmarshal(gotBody, &sent); err != nil {
		t.Fatalf("decode agent body: %v (%s)", err, gotBody)
	}
	if len(sent.Messages) != 3 {
		t.Fatalf("want [summary, old-3, new-turn], got %+v", sent.Messages)
	}
	if sent.Messages[0].ID != "" || sent.Messages[0].Content != summaryFrame+"SUM" {
		t.Fatalf("summary turn wrong: %+v", sent.Messages[0])
	}
	if sent.Messages[1].Content != "old-3" || sent.Messages[1].ID == "" {
		t.Fatalf("tail should be id-tagged old-3, got %+v", sent.Messages[1])
	}
	if sent.Messages[2].Content != "new-turn" {
		t.Fatalf("last should be the new turn, got %+v", sent.Messages[2])
	}
}

// compactionEndFrame builds a compaction node_end carrying the watermark id.
func compactionEndFrame(id, wid string) string {
	return fmt.Sprintf("event: node_end\ndata: {\"id\":%q,\"output\":{\"summary_through_id\":%q}}\n\n", id, wid)
}

func TestSend_PersistsCompactionSummary(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	ta, uid := signup(t, nil, "a@x.com")
	var cid int64
	testPool.QueryRow(ctx, `insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid)
	var mid int64
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'user','old') returning id`, cid).Scan(&mid)

	client := fakeAgent(t, http.StatusOK,
		nodeStartFrame("s1:compaction", "", "compaction", "", ""),
		deltaFrame("s1:compaction", "RECAP"),
		compactionEndFrame("s1:compaction", strconv.FormatInt(mid, 10)),
		textFrames("a", "done"), endFrame)
	mux := newTestMux(client)

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	var sum *string
	var wm *int64
	testPool.QueryRow(ctx, `select summary, summary_through_message_id from conversations where id=$1`, cid).Scan(&sum, &wm)
	if sum == nil || *sum != "RECAP" || wm == nil || *wm != mid {
		t.Fatalf("summary=%v watermark=%v (want RECAP, %d)", sum, wm, mid)
	}
}

func TestSend_NullWatermarkNotPersisted(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	ta, uid := signup(t, nil, "a@x.com")
	var cid int64
	testPool.QueryRow(ctx, `insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid)

	nullEnd := "event: node_end\ndata: {\"id\":\"s1:compaction\",\"output\":{\"summary_through_id\":null}}\n\n"
	client := fakeAgent(t, http.StatusOK,
		nodeStartFrame("s1:compaction", "", "compaction", "", ""),
		deltaFrame("s1:compaction", "RECAP"), nullEnd,
		textFrames("a", "done"), endFrame)
	mux := newTestMux(client)

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	var sum *string
	testPool.QueryRow(ctx, `select summary from conversations where id=$1`, cid).Scan(&sum)
	if sum != nil {
		t.Fatalf("null id must not persist, got %q", *sum)
	}
}

func TestSend_LastNonNullCompactionWins(t *testing.T) {
	resetDB(t)
	ctx := context.Background()
	ta, uid := signup(t, nil, "a@x.com")
	var cid int64
	testPool.QueryRow(ctx, `insert into conversations (user_id) values ($1) returning id`, uid).Scan(&cid)
	var m1, m2 int64
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'user','one') returning id`, cid).Scan(&m1)
	testPool.QueryRow(ctx, `insert into messages (conversation_id, role, content) values ($1,'assistant','two') returning id`, cid).Scan(&m2)

	client := fakeAgent(t, http.StatusOK,
		nodeStartFrame("s1:compaction", "", "compaction", "", ""),
		deltaFrame("s1:compaction", "FIRST"), compactionEndFrame("s1:compaction", strconv.FormatInt(m1, 10)),
		nodeStartFrame("s2:compaction", "", "compaction", "", ""),
		deltaFrame("s2:compaction", "SECOND"), compactionEndFrame("s2:compaction", strconv.FormatInt(m2, 10)),
		textFrames("a", "done"), endFrame)
	mux := newTestMux(client)

	do(t, mux, http.MethodPost, fmt.Sprintf("/api/conversations/%d/messages", cid), ta,
		map[string]string{"content": "hi"})

	var sum *string
	var wm *int64
	testPool.QueryRow(ctx, `select summary, summary_through_message_id from conversations where id=$1`, cid).Scan(&sum, &wm)
	if sum == nil || *sum != "SECOND" || wm == nil || *wm != m2 {
		t.Fatalf("last should win: summary=%v watermark=%v (want SECOND, %d)", sum, wm, m2)
	}
}
