// Package ai provides AI-powered features using Docker Model Runner.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the default Docker Model Runner endpoint.
	DefaultBaseURL = "http://localhost:12434"

	// DefaultModel is the default model for commit message generation.
	// Using the specific quantized version that's commonly pulled.
	DefaultModel = "ai/smollm2:360M-Q4_K_M"

	// DefaultTimeout is the default timeout for API calls.
	DefaultTimeout = 5 * time.Second

	// MaxCommitLength is the maximum length of a commit message.
	MaxCommitLength = 72

	// MaxDiffLength is the maximum length of diff to send to the model.
	MaxDiffLength = 2000
)

// CommitGenerator generates commit messages using Docker Model Runner.
type CommitGenerator struct {
	client  *http.Client
	baseURL string
	model   string
	timeout time.Duration
}

// Option configures a CommitGenerator.
type Option func(*CommitGenerator)

// WithBaseURL sets the base URL for Docker Model Runner.
func WithBaseURL(url string) Option {
	return func(g *CommitGenerator) {
		g.baseURL = url
	}
}

// WithModel sets the model to use for generation.
func WithModel(model string) Option {
	return func(g *CommitGenerator) {
		g.model = model
	}
}

// WithTimeout sets the timeout for API calls.
func WithTimeout(timeout time.Duration) Option {
	return func(g *CommitGenerator) {
		g.timeout = timeout
	}
}

// NewCommitGenerator creates a new CommitGenerator with the given options.
func NewCommitGenerator(opts ...Option) *CommitGenerator {
	g := &CommitGenerator{
		baseURL: DefaultBaseURL,
		model:   DefaultModel,
		timeout: DefaultTimeout,
	}

	for _, opt := range opts {
		opt(g)
	}

	g.client = &http.Client{
		Timeout: g.timeout,
	}

	return g
}

// ChatCompletionRequest represents an OpenAI-compatible chat completion request.
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse represents an OpenAI-compatible chat completion response.
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
}

// Choice represents a completion choice.
type Choice struct {
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
}

// Generate creates a commit message based on the changed files and diff.
func (g *CommitGenerator) Generate(ctx context.Context, changedFiles []string, diff string) (string, error) {
	prompt := buildPrompt(changedFiles, diff)

	req := ChatCompletionRequest{
		Model: g.model,
		Messages: []Message{
			{
				Role: "system",
				Content: `You are a commit message generator. Generate a concise git commit message.
Rules:
- One line only, max 72 characters
- Start with a verb (Add, Update, Remove, Fix, Refactor, etc.)
- Be specific about what changed
- No period at the end
- No quotes around the message`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		MaxTokens:   100,
		Temperature: 0.3,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := g.baseURL + "/engines/llama.cpp/v1/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to call Docker Model Runner: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docker Model Runner returned status %d", resp.StatusCode)
	}

	var result ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	msg := strings.TrimSpace(result.Choices[0].Message.Content)

	// Remove surrounding quotes if present (models sometimes add them)
	msg = strings.Trim(msg, `"'`)

	// Truncate if too long
	if len(msg) > MaxCommitLength {
		msg = msg[:MaxCommitLength]
	}

	return msg, nil
}

// IsAvailable checks if Docker Model Runner is available and responding.
func (g *CommitGenerator) IsAvailable(ctx context.Context) bool {
	url := g.baseURL + "/models"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

// buildPrompt creates the prompt for the model.
func buildPrompt(changedFiles []string, diff string) string {
	// Truncate diff if too long
	if len(diff) > MaxDiffLength {
		diff = diff[:MaxDiffLength] + "\n... (truncated)"
	}

	var sb strings.Builder
	sb.WriteString("Generate a commit message for these changes:\n\n")
	sb.WriteString("Changed files:\n")
	for _, f := range changedFiles {
		sb.WriteString("- ")
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	sb.WriteString("\nDiff:\n")
	sb.WriteString(diff)
	sb.WriteString("\n\nCommit message:")

	return sb.String()
}
