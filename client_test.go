package veniceai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	veniceai "github.com/13rac1/veniceai-go"
)

// modelsJSON is a minimal valid response for the /models endpoint.
var modelsJSON = map[string]any{"object": "list", "data": []any{}}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(modelsJSON); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
}

func TestNewClient(t *testing.T) {
	client, err := veniceai.NewClient("test-key")
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}
	if client.API == nil {
		t.Fatal("API client is nil")
	}
}

func TestWithBaseURL(t *testing.T) {
	var called bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(modelsJSON); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("test-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.API.ListModelsWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	if !called {
		t.Error("request was not routed to the custom base URL")
	}
}

func TestWithHTTPClient(t *testing.T) {
	var called bool
	ts := newTestServer(t)
	defer ts.Close()

	custom := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			called = true
			return http.DefaultTransport.RoundTrip(req)
		}),
	}

	client, err := veniceai.NewClient("test-key",
		veniceai.WithBaseURL(ts.URL),
		veniceai.WithHTTPClient(custom),
	)
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.API.ListModelsWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}
	if !called {
		t.Error("custom HTTP client transport was not used")
	}
}

func TestBearerTokenSent(t *testing.T) {
	var receivedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(modelsJSON); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}))
	defer ts.Close()

	client, err := veniceai.NewClient("my-secret-key", veniceai.WithBaseURL(ts.URL))
	if err != nil {
		t.Fatalf("NewClient() error: %v", err)
	}

	_, err = client.API.ListModelsWithResponse(t.Context(), nil)
	if err != nil {
		t.Fatalf("ListModels() error: %v", err)
	}

	if receivedAuth != "Bearer my-secret-key" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer my-secret-key")
	}
}

// roundTripFunc adapts a function to the http.RoundTripper interface.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
