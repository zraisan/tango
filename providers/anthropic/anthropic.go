package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/zraisan/gotique/models"
)

const (
	apiVersion       = "2023-06-01"
	defaultMaxTokens = 16000
)

type Provider struct {
	endpoint string
	apiKey   string
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
}

func New(apiKey string, endpoint ...string) *Provider {
	providerEndpoint := "https://api.anthropic.com/v1/messages"
	if len(endpoint) > 0 && endpoint[0] != "" {
		providerEndpoint = endpoint[0]
	}

	return &Provider{
		endpoint: providerEndpoint,
		apiKey:   apiKey,
	}
}

func (p *Provider) Generate(ctx context.Context, modelReq models.Request) (*models.Response, error) {
	systemParts := []string{}
	if modelReq.SystemPrompt != "" {
		systemParts = append(systemParts, modelReq.SystemPrompt)
	}

	systemParts = append(systemParts, modelReq.Instructions...)

	messages := make([]message, 0, len(modelReq.History)+1)
	for _, msg := range modelReq.History {
		switch msg.Role {
		case models.RoleSystem:
			systemParts = append(systemParts, msg.Content)
		case models.RoleUser, models.RoleAssistant:
			messages = appendMessage(messages, string(msg.Role), msg.Content)
		default:
			return nil, fmt.Errorf("anthropic: unsupported role %q", msg.Role)
		}
	}

	messages = appendMessage(messages, string(models.RoleUser), modelReq.Input)

	body := request{
		Model:     modelReq.Model,
		MaxTokens: defaultMaxTokens,
		System:    strings.Join(systemParts, "\n"),
		Messages:  messages,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", apiVersion)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("anthropic request failed: %s: %s", resp.Status, string(respBody))
	}

	var result response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if result.StopReason == "refusal" {
		return nil, fmt.Errorf("anthropic request refused: %s", string(respBody))
	}

	return &models.Response{
		Text: result.text(),
	}, nil
}

func appendMessage(messages []message, role, content string) []message {
	if n := len(messages); n > 0 && messages[n-1].Role == role {
		messages[n-1].Content += "\n\n" + content
		return messages
	}

	return append(messages, message{Role: role, Content: content})
}

func (r *response) text() string {
	var parts []string
	for _, content := range r.Content {
		if content.Type == "text" {
			parts = append(parts, content.Text)
		}
	}

	return strings.Join(parts, "")
}
