package gemini

import (
		"context"
		"fmt"
		"sync"

		"github.com/google/generative-ai-go/genai"
		"google.golang.org/api/option"
	)

const (
		ModelName = "gemini-2.5-flash"
	)

// Client wraps the official Gemini SDK client.
type Client struct {
		apiKey    string
		genClient *genai.Client
		model     *genai.GenerativeModel
		mu        sync.Mutex
}

// NewClient creates a new Gemini client using the official SDK.
func NewClient(apiKey string) (*Client, error) {
		if apiKey == "" {
					return nil, fmt.Errorf("gemini_api_key not configured in config.yaml")
				}

		ctx := context.Background()
		genClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
		if err != nil {
					return nil, fmt.Errorf("failed to create Gemini client: %w", err)
				}

		model := genClient.GenerativeModel(ModelName)

		return &Client{
					apiKey:    apiKey,
					genClient: genClient,
					model:     model,
				}, nil
}

// SendPrompt sends a single prompt (stateless, no session context).
func (c *Client) SendPrompt(prompt string) (string, error) {
		ctx := context.Background()

		resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
		if err != nil {
					return "", fmt.Errorf("gemini API error: %w", err)
				}

		return extractText(resp), nil
}

// ChatSession wraps a Gemini chat session for multi-turn conversations
// that preserve context across messages.
type ChatSession struct {
		session *genai.ChatSession
		mu      sync.Mutex
}

// StartChat creates a new chat session that preserves context across messages.
func (c *Client) StartChat() *ChatSession {
		return &ChatSession{
					session: c.model.StartChat(),
				}
}

// Send sends a message in the chat session, preserving conversation history.
func (cs *ChatSession) Send(prompt string) (string, error) {
		cs.mu.Lock()
		defer cs.mu.Unlock()

		ctx := context.Background()
		resp, err := cs.session.SendMessage(ctx, genai.Text(prompt))
		if err != nil {
					return "", fmt.Errorf("gemini chat error: %w", err)
				}

		return extractText(resp), nil
}

// GetHistory returns the conversation history of this chat session.
func (cs *ChatSession) GetHistory() []*genai.Content {
		cs.mu.Lock()
		defer cs.mu.Unlock()
		return cs.session.History
}

// Close releases resources held by the Gemini client.
func (c *Client) Close() error {
		return c.genClient.Close()
}

// extractText extracts the text response from a Gemini API response.
func extractText(resp *genai.GenerateContentResponse) string {
		if resp == nil || len(resp.Candidates) == 0 {
					return ""
				}

		candidate := resp.Candidates[0]
		if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
					return ""
				}

		var result string
		for _, part := range candidate.Content.Parts {
					if text, ok := part.(genai.Text); ok {
									result += string(text)
								}
				}
		return result
}
