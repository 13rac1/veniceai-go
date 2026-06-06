package veniceai_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/13rac1/veniceai-go"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// chatCompletionJSON is a minimal valid chat completion response.
const chatCompletionJSON = `{"id":"chatcmpl-123","object":"chat.completion","created":1234567890,"model":"llama-3.3-70b","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`

// veniceHeaders sets common Venice response headers on w.
func veniceHeaders(w http.ResponseWriter) {
	w.Header().Set("x-venice-version", "test-v1")
	w.Header().Set("x-venice-balance-diem", "42.5")
	w.Header().Set("x-venice-balance-usd", "10.25")
	w.Header().Set("x-ratelimit-limit-requests", "100")
	w.Header().Set("x-ratelimit-remaining-requests", "99")
	w.Header().Set("x-ratelimit-reset-requests", "1780000000")
	w.Header().Set("x-ratelimit-limit-tokens", "2000000")
	w.Header().Set("x-ratelimit-remaining-tokens", "1999000")
	w.Header().Set("x-ratelimit-reset-tokens", "1780000001")
	w.Header().Set("x-venice-model-id", "llama-3.3-70b")
	w.Header().Set("x-venice-model-name", "Llama 3.3 70B")
	w.Header().Set("cf-ray", "abc123")
}

func writeChatCompletion(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(chatCompletionJSON)); err != nil {
		t.Errorf("writeChatCompletion: %v", err)
	}
}

func writeSSE(t *testing.T, w http.ResponseWriter, events ...string) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	flusher, ok := w.(http.Flusher)
	if !ok {
		t.Fatal("ResponseWriter does not support Flusher")
	}
	for _, event := range events {
		if _, err := io.WriteString(w, event); err != nil {
			t.Errorf("writeSSE: %v", err)
		}
	}
	flusher.Flush()
}

func chatParams() *openai.ChatCompletionNewParams {
	return &openai.ChatCompletionNewParams{
		Model: "llama-3.3-70b",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("hi"),
		},
	}
}

func TestChatComplete(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		veniceHeaders(w)
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}
	if result.Choices[0].Message.Content != "hello" {
		t.Errorf("content = %q, want %q", result.Choices[0].Message.Content, "hello")
	}
}

func TestChatCompleteResponseHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		veniceHeaders(w)
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	h := result.Headers
	if h.Version != "test-v1" {
		t.Errorf("Version = %q, want %q", h.Version, "test-v1")
	}
	if h.BalanceDiem != "42.5" {
		t.Errorf("BalanceDiem = %q, want %q", h.BalanceDiem, "42.5")
	}
	if h.BalanceUSD != "10.25" {
		t.Errorf("BalanceUSD = %q, want %q", h.BalanceUSD, "10.25")
	}
	if h.RateLimitRequests != 100 {
		t.Errorf("RateLimitRequests = %d, want 100", h.RateLimitRequests)
	}
	if h.RateLimitRequestsRemaining != 99 {
		t.Errorf("RateLimitRequestsRemaining = %d, want 99", h.RateLimitRequestsRemaining)
	}
	if h.RateLimitTokens != 2000000 {
		t.Errorf("RateLimitTokens = %d, want 2000000", h.RateLimitTokens)
	}
	if h.ModelID != "llama-3.3-70b" {
		t.Errorf("ModelID = %q, want %q", h.ModelID, "llama-3.3-70b")
	}
	if h.ModelName != "Llama 3.3 70B" {
		t.Errorf("ModelName = %q, want %q", h.ModelName, "Llama 3.3 70B")
	}
	if h.CFRay != "abc123" {
		t.Errorf("CFRay = %q, want %q", h.CFRay, "abc123")
	}
}

func TestChatCompleteVeniceParams(t *testing.T) {
	var receivedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ws := veniceai.WebSearchOn
	_, err = client.ChatComplete(t.Context(), chatParams(), &veniceai.VeniceParameters{
		EnableWebSearch:           &ws,
		IncludeVeniceSystemPrompt: veniceai.Ptr(false),
	})
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	vp, ok := body["venice_parameters"].(map[string]any)
	if !ok {
		t.Fatalf("venice_parameters not found in body: %s", receivedBody)
	}
	if vp["enable_web_search"] != "on" {
		t.Errorf("enable_web_search = %v, want %q", vp["enable_web_search"], "on")
	}
	if vp["include_venice_system_prompt"] != false {
		t.Errorf("include_venice_system_prompt = %v, want false", vp["include_venice_system_prompt"])
	}
}

func TestChatCompleteNilVeniceParams(t *testing.T) {
	var receivedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	if _, ok := body["venice_parameters"]; ok {
		t.Errorf("venice_parameters should not be in body when nil, got: %s", receivedBody)
	}
}

func TestChatCompleteContentSafetyHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-venice-is-content-violation", "true")
		w.Header().Set("x-venice-is-blurred", "true")
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	if !result.Headers.IsContentViolation {
		t.Error("IsContentViolation = false, want true")
	}
	if !result.Headers.IsBlurred {
		t.Error("IsBlurred = false, want true")
	}
}

func TestChatCompleteDeprecationHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-venice-model-deprecation-warning", "This model will be removed")
		w.Header().Set("x-venice-model-deprecation-date", "2026-07-01T00:00:00Z")
		w.Header().Set("x-venice-deprecated-replacement", "llama-4-70b")
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	h := result.Headers
	if h.DeprecationWarning != "This model will be removed" {
		t.Errorf("DeprecationWarning = %q, want %q", h.DeprecationWarning, "This model will be removed")
	}
	if h.DeprecationDate != "2026-07-01T00:00:00Z" {
		t.Errorf("DeprecationDate = %q, want %q", h.DeprecationDate, "2026-07-01T00:00:00Z")
	}
	if h.DeprecatedReplacement != "llama-4-70b" {
		t.Errorf("DeprecatedReplacement = %q, want %q", h.DeprecatedReplacement, "llama-4-70b")
	}
}

func TestChatCompleteStreamResponseHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		veniceHeaders(w)
		writeSSE(t, w,
			"data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"created\":0,\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n",
			"data: [DONE]\n\n",
		)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream := client.ChatCompleteStream(t.Context(), chatParams(), nil)
	defer stream.Close()

	if stream.Headers.Version != "test-v1" {
		t.Errorf("Version = %q, want %q", stream.Headers.Version, "test-v1")
	}
	if stream.Headers.BalanceDiem != "42.5" {
		t.Errorf("BalanceDiem = %q, want %q", stream.Headers.BalanceDiem, "42.5")
	}
	if stream.Headers.RateLimitRequests != 100 {
		t.Errorf("RateLimitRequests = %d, want 100", stream.Headers.RateLimitRequests)
	}
	if stream.Headers.CFRay != "abc123" {
		t.Errorf("CFRay = %q, want %q", stream.Headers.CFRay, "abc123")
	}

	var buf strings.Builder
	for stream.Next() {
		buf.WriteString(stream.Current().Choices[0].Delta.Content)
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}
	if buf.String() != "hi" {
		t.Errorf("streamed content = %q, want %q", buf.String(), "hi")
	}
}

func TestChatCompleteStreamVeniceParams(t *testing.T) {
	var receivedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		receivedBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
		}
		writeSSE(t, w, "data: [DONE]\n\n")
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	stream := client.ChatCompleteStream(t.Context(), chatParams(), &veniceai.VeniceParameters{
		DisableThinking: veniceai.Ptr(true),
	})
	defer stream.Close()
	for stream.Next() {
	}
	if err := stream.Err(); err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(receivedBody, &body); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}

	vp, ok := body["venice_parameters"].(map[string]any)
	if !ok {
		t.Fatalf("venice_parameters not found in body: %s", receivedBody)
	}
	if vp["disable_thinking"] != true {
		t.Errorf("disable_thinking = %v, want true", vp["disable_thinking"])
	}
}

func TestChatCompletePassesRequestOptions(t *testing.T) {
	var receivedHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("X-Custom")
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ChatComplete(t.Context(), chatParams(), nil,
		option.WithHeader("X-Custom", "test-value"),
	)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}
	if receivedHeader != "test-value" {
		t.Errorf("X-Custom = %q, want %q", receivedHeader, "test-value")
	}
}

func TestChatCompleteMalformedHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("x-ratelimit-limit-requests", "not-a-number")
		w.Header().Set("x-ratelimit-reset-requests", "also-bad")
		w.Header().Set("x-venice-is-content-violation", "maybe")
		writeChatCompletion(t, w)
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	result, err := client.ChatComplete(t.Context(), chatParams(), nil)
	if err != nil {
		t.Fatalf("ChatComplete: %v", err)
	}

	if result.Headers.RateLimitRequests != 0 {
		t.Errorf("RateLimitRequests = %d, want 0 for malformed header", result.Headers.RateLimitRequests)
	}
	if result.Headers.RateLimitRequestsReset != 0 {
		t.Errorf("RateLimitRequestsReset = %d, want 0 for malformed header", result.Headers.RateLimitRequestsReset)
	}
	if result.Headers.IsContentViolation {
		t.Error("IsContentViolation = true, want false for non-'true' value")
	}
}

func TestPtr(t *testing.T) {
	b := veniceai.Ptr(true)
	if !*b {
		t.Error("Ptr(true) = false, want true")
	}
	s := veniceai.Ptr("hello")
	if *s != "hello" {
		t.Errorf("Ptr(\"hello\") = %v, want \"hello\"", *s)
	}
	ws := veniceai.Ptr(veniceai.WebSearchOn)
	if *ws != veniceai.WebSearchOn {
		t.Errorf("Ptr(WebSearchOn) = %v, want %q", *ws, veniceai.WebSearchOn)
	}
}
