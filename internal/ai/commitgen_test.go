package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommitGenerator_Generate_Success(t *testing.T) {
	t.Parallel()

	// Create a mock server that simulates Docker Model Runner
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.URL.Path != "/engines/llama.cpp/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}

		// Parse request body to verify it's correct
		var req ChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		// Verify the request has the expected structure
		if req.Model != "ai/smollm2:360M-Q4_K_M" {
			t.Errorf("unexpected model: %s", req.Model)
		}
		if len(req.Messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(req.Messages))
		}

		// Return a successful response
		resp := ChatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   "ai/smollm2",
			Choices: []Choice{
				{
					Index: 0,
					Message: Message{
						Role:    "assistant",
						Content: "Update MCP server configuration",
					},
					FinishReason: "stop",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	gen := NewCommitGenerator(
		WithBaseURL(server.URL),
		WithTimeout(5*time.Second),
	)

	ctx := context.Background()
	changedFiles := []string{"mcp-servers.json"}
	diff := `diff --git a/mcp-servers.json b/mcp-servers.json
+  "postgres": {
+    "command": "npx",
+    "args": ["-y", "@modelcontextprotocol/server-postgres"]
+  }`

	msg, err := gen.Generate(ctx, changedFiles, diff)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	expected := "Update MCP server configuration"
	if msg != expected {
		t.Errorf("Generate() = %q, want %q", msg, expected)
	}
}

func TestCommitGenerator_Generate_FallbackOnError(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	gen := NewCommitGenerator(
		WithBaseURL(server.URL),
		WithTimeout(1*time.Second),
	)

	ctx := context.Background()
	_, err := gen.Generate(ctx, []string{"settings.json"}, "some diff")

	// Should return an error (caller decides fallback behavior)
	if err == nil {
		t.Error("Generate() expected error, got nil")
	}
}

func TestCommitGenerator_Generate_FallbackOnTimeout(t *testing.T) {
	t.Parallel()

	// Create a mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Second) // Longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gen := NewCommitGenerator(
		WithBaseURL(server.URL),
		WithTimeout(100*time.Millisecond), // Very short timeout
	)

	ctx := context.Background()
	_, err := gen.Generate(ctx, []string{"settings.json"}, "some diff")

	// Should return a timeout error
	if err == nil {
		t.Error("Generate() expected timeout error, got nil")
	}
}

func TestCommitGenerator_Generate_TrimsResponse(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns response with extra whitespace
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Content: "  Add new config file  \n",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	gen := NewCommitGenerator(WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := gen.Generate(ctx, []string{"file.txt"}, "diff")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	expected := "Add new config file"
	if msg != expected {
		t.Errorf("Generate() = %q, want %q", msg, expected)
	}
}

func TestCommitGenerator_Generate_TruncatesLongMessages(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns a very long message
	longMessage := strings.Repeat("A", 100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Content: longMessage,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	gen := NewCommitGenerator(WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := gen.Generate(ctx, []string{"file.txt"}, "diff")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// Should be truncated to MaxCommitLength (72)
	if len(msg) > MaxCommitLength {
		t.Errorf("Generate() message too long: %d > %d", len(msg), MaxCommitLength)
	}
}

func TestCommitGenerator_Generate_EmptyChoices(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns empty choices
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []Choice{},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	gen := NewCommitGenerator(WithBaseURL(server.URL))
	ctx := context.Background()

	_, err := gen.Generate(ctx, []string{"file.txt"}, "diff")

	// Should return an error when no choices
	if err == nil {
		t.Error("Generate() expected error for empty choices, got nil")
	}
}

func TestCommitGenerator_IsAvailable_True(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns models
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[{"namespace":"ai","name":"smollm2"}]`))
	}))
	defer server.Close()

	gen := NewCommitGenerator(WithBaseURL(server.URL))
	ctx := context.Background()

	if !gen.IsAvailable(ctx) {
		t.Error("IsAvailable() = false, want true")
	}
}

func TestCommitGenerator_IsAvailable_False(t *testing.T) {
	t.Parallel()

	// Use an invalid URL to simulate DMR not running
	gen := NewCommitGenerator(WithBaseURL("http://localhost:99999"))
	ctx := context.Background()

	if gen.IsAvailable(ctx) {
		t.Error("IsAvailable() = true, want false")
	}
}

func TestCommitGenerator_DefaultConfig(t *testing.T) {
	t.Parallel()

	gen := NewCommitGenerator()

	if gen.baseURL != DefaultBaseURL {
		t.Errorf("baseURL = %q, want %q", gen.baseURL, DefaultBaseURL)
	}
	if gen.model != DefaultModel {
		t.Errorf("model = %q, want %q", gen.model, DefaultModel)
	}
	if gen.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", gen.timeout, DefaultTimeout)
	}
}

func TestCommitGenerator_Generate_StripsQuotes(t *testing.T) {
	t.Parallel()

	// Create a mock server that returns response with quotes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := ChatCompletionResponse{
			Choices: []Choice{
				{
					Message: Message{
						Content: `"Add new config file"`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer server.Close()

	gen := NewCommitGenerator(WithBaseURL(server.URL))
	ctx := context.Background()

	msg, err := gen.Generate(ctx, []string{"file.txt"}, "diff")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	expected := "Add new config file"
	if msg != expected {
		t.Errorf("Generate() = %q, want %q", msg, expected)
	}
}

func TestBuildPrompt(t *testing.T) {
	t.Parallel()

	changedFiles := []string{"settings.json", "mcp-servers.json"}
	diff := `+  "theme": "dark"`

	prompt := buildPrompt(changedFiles, diff)

	// Should contain changed files
	if !strings.Contains(prompt, "settings.json") {
		t.Error("prompt should contain file names")
	}

	// Should contain diff
	if !strings.Contains(prompt, `"theme": "dark"`) {
		t.Error("prompt should contain diff")
	}
}

func TestBuildPrompt_TruncatesDiff(t *testing.T) {
	t.Parallel()

	// Create a very long diff
	longDiff := strings.Repeat("x", MaxDiffLength+1000)
	prompt := buildPrompt([]string{"file.txt"}, longDiff)

	// Prompt should not be excessively long
	if len(prompt) > MaxDiffLength+500 { // Some overhead for the template
		t.Errorf("prompt too long: %d", len(prompt))
	}

	// Should indicate truncation
	if !strings.Contains(prompt, "... (truncated)") {
		t.Error("prompt should indicate truncation")
	}
}
