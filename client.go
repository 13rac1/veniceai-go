package veniceai

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/13rac1/veniceai-go/api"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// DefaultBaseURL is the production Venice.ai API base URL.
const DefaultBaseURL = "https://api.venice.ai/api/v1"

// Client provides access to the Venice.ai API.
//
// OpenAI-compatible endpoints (chat completions, embeddings, audio, images,
// models) are available through the OpenAI field. All Venice API endpoints
// are available through the API field.
type Client struct {
	// OpenAI provides access to OpenAI-compatible endpoints with streaming,
	// retries, and rich types via the openai-go library.
	OpenAI openai.Client

	// API provides access to all Venice API endpoints via the generated client.
	API *api.ClientWithResponses
}

type clientConfig struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a [Client].
type Option func(*clientConfig)

// WithBaseURL overrides the default Venice API base URL.
func WithBaseURL(url string) Option {
	return func(c *clientConfig) {
		c.baseURL = url
	}
}

// WithHTTPClient sets the HTTP client used for API requests.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *clientConfig) {
		c.httpClient = httpClient
	}
}

// NewClient creates a Venice.ai API client authenticated with the given API key.
func NewClient(apiKey string, opts ...Option) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("veniceai: API key must not be empty")
	}

	cfg := &clientConfig{
		baseURL:    DefaultBaseURL,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	// Build generated Venice API client with Bearer auth.
	apiClient, err := api.NewClientWithResponses(cfg.baseURL,
		api.WithHTTPClient(cfg.httpClient),
		api.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			req.Header.Set("Authorization", "Bearer "+apiKey)
			return nil
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("veniceai: creating API client: %w", err)
	}

	return &Client{
		OpenAI: openai.NewClient(
			option.WithAPIKey(apiKey),
			option.WithBaseURL(cfg.baseURL),
			option.WithHTTPClient(cfg.httpClient),
		),
		API: apiClient,
	}, nil
}
