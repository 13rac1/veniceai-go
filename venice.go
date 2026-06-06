package veniceai

import (
	"context"
	"net/http"
	"strconv"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/ssestream"
)

// WebSearch controls web search behavior for a chat completion request.
type WebSearch string

// Valid values for [WebSearch].
const (
	WebSearchAuto WebSearch = "auto"
	WebSearchOff  WebSearch = "off"
	WebSearchOn   WebSearch = "on"
)

// VeniceParameters are Venice-specific parameters for chat completion requests.
// These extend the standard OpenAI chat completion API with Venice features.
type VeniceParameters struct {
	CharacterSlug                  *string    `json:"character_slug,omitempty"`
	StripThinkingResponse          *bool      `json:"strip_thinking_response,omitempty"`
	DisableThinking                *bool      `json:"disable_thinking,omitempty"`
	EnableE2ee                     *bool      `json:"enable_e2ee,omitempty"`
	EnableWebSearch                *WebSearch `json:"enable_web_search,omitempty"`
	EnableWebScraping              *bool      `json:"enable_web_scraping,omitempty"`
	EnableWebCitations             *bool      `json:"enable_web_citations,omitempty"`
	EnableXSearch                  *bool      `json:"enable_x_search,omitempty"`
	IncludeSearchResultsInStream   *bool      `json:"include_search_results_in_stream,omitempty"`
	ReturnSearchResultsAsDocuments *bool      `json:"return_search_results_as_documents,omitempty"`
	IncludeVeniceSystemPrompt      *bool      `json:"include_venice_system_prompt,omitempty"`
}

// Ptr returns a pointer to v. It is a convenience for setting optional fields
// on [VeniceParameters].
func Ptr[T any](v T) *T { return &v }

// ResponseHeaders contains Venice-specific HTTP response headers returned by
// inference endpoints.
type ResponseHeaders struct {
	// Version is the Venice API server version (x-venice-version).
	Version string
	// BalanceDiem is the DIEM token balance (x-venice-balance-diem).
	BalanceDiem string
	// BalanceUSD is the USD credit balance (x-venice-balance-usd).
	BalanceUSD string

	// RateLimitRequests is the max requests allowed in the current window.
	RateLimitRequests int
	// RateLimitRequestsRemaining is the requests remaining in the current window.
	RateLimitRequestsRemaining int
	// RateLimitRequestsReset is the unix timestamp (ms) when the request limit resets.
	RateLimitRequestsReset int64
	// RateLimitTokens is the max tokens allowed in the current window.
	RateLimitTokens int
	// RateLimitTokensRemaining is the tokens remaining in the current window.
	RateLimitTokensRemaining int
	// RateLimitTokensReset is the unix timestamp (ms) when the token limit resets.
	RateLimitTokensReset int64

	// ModelID is the model identifier used for the request (x-venice-model-id).
	ModelID string
	// ModelName is the friendly name of the model (x-venice-model-name).
	ModelName string
	// DeprecationWarning is a warning that the model is scheduled for deprecation.
	DeprecationWarning string
	// DeprecationDate is the ISO 8601 date when the model will be removed.
	DeprecationDate string
	// DeprecatedReplacement is the model ID to migrate to.
	DeprecatedReplacement string

	// IsContentViolation indicates whether the content violated Venice ToS.
	IsContentViolation bool
	// IsBlurred indicates whether a generated image was blurred.
	IsBlurred bool

	// CFRay is the Cloudflare request ID for troubleshooting.
	CFRay string
}

func parseResponseHeaders(h http.Header) ResponseHeaders {
	return ResponseHeaders{
		Version:     h.Get("x-venice-version"),
		BalanceDiem: h.Get("x-venice-balance-diem"),
		BalanceUSD:  h.Get("x-venice-balance-usd"),

		RateLimitRequests:          headerInt(h, "x-ratelimit-limit-requests"),
		RateLimitRequestsRemaining: headerInt(h, "x-ratelimit-remaining-requests"),
		RateLimitRequestsReset:     headerInt64(h, "x-ratelimit-reset-requests"),
		RateLimitTokens:            headerInt(h, "x-ratelimit-limit-tokens"),
		RateLimitTokensRemaining:   headerInt(h, "x-ratelimit-remaining-tokens"),
		RateLimitTokensReset:       headerInt64(h, "x-ratelimit-reset-tokens"),

		ModelID:               h.Get("x-venice-model-id"),
		ModelName:             h.Get("x-venice-model-name"),
		DeprecationWarning:    h.Get("x-venice-model-deprecation-warning"),
		DeprecationDate:       h.Get("x-venice-model-deprecation-date"),
		DeprecatedReplacement: h.Get("x-venice-deprecated-replacement"),

		IsContentViolation: h.Get("x-venice-is-content-violation") == "true",
		IsBlurred:          h.Get("x-venice-is-blurred") == "true",

		CFRay: h.Get("cf-ray"),
	}
}

// headerInt returns the named header as an integer, or 0 if absent or unparseable.
func headerInt(h http.Header, key string) int {
	s := h.Get(key)
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return v
}

// headerInt64 returns the named header as an int64, or 0 if absent or unparseable.
func headerInt64(h http.Header, key string) int64 {
	s := h.Get(key)
	if s == "" {
		return 0
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// ChatCompletion wraps an OpenAI chat completion response with Venice-specific
// response headers.
type ChatCompletion struct {
	*openai.ChatCompletion
	Headers ResponseHeaders
}

// ChatComplete creates a chat completion with support for Venice-specific
// parameters and response headers. It wraps the openai-go Chat.Completions.New
// method, preserving retries and all openai-go options.
//
// Pass nil for veniceParams to make a standard OpenAI-compatible request.
func (c *Client) ChatComplete(
	ctx context.Context,
	params *openai.ChatCompletionNewParams,
	veniceParams *VeniceParameters,
	opts ...option.RequestOption,
) (*ChatCompletion, error) {
	var raw *http.Response
	opts = append(opts, option.WithResponseInto(&raw))
	if veniceParams != nil {
		opts = append(opts, option.WithJSONSet("venice_parameters", veniceParams))
	}

	completion, err := c.OpenAI.Chat.Completions.New(ctx, *params, opts...)
	if err != nil {
		return nil, err
	}
	return &ChatCompletion{
		ChatCompletion: completion,
		Headers:        parseResponseHeaders(raw.Header),
	}, nil
}

// ChatCompletionStream wraps an openai-go SSE stream with Venice-specific
// response headers. Iterate with Next/Current/Err as usual.
type ChatCompletionStream struct {
	*ssestream.Stream[openai.ChatCompletionChunk]
	Headers ResponseHeaders
}

// ChatCompleteStream creates a streaming chat completion with support for
// Venice-specific parameters and response headers. It wraps the openai-go
// Chat.Completions.NewStreaming method.
//
// Pass nil for veniceParams to make a standard OpenAI-compatible request.
// Response headers are available immediately on the returned stream before
// iteration begins.
func (c *Client) ChatCompleteStream(
	ctx context.Context,
	params *openai.ChatCompletionNewParams,
	veniceParams *VeniceParameters,
	opts ...option.RequestOption,
) *ChatCompletionStream {
	var raw *http.Response
	opts = append(opts, option.WithResponseInto(&raw))
	if veniceParams != nil {
		opts = append(opts, option.WithJSONSet("venice_parameters", veniceParams))
	}

	stream := c.OpenAI.Chat.Completions.NewStreaming(ctx, *params, opts...)
	var headers ResponseHeaders
	if raw != nil {
		headers = parseResponseHeaders(raw.Header)
	}
	return &ChatCompletionStream{
		Stream:  stream,
		Headers: headers,
	}
}
