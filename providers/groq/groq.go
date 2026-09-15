package openailike

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

type Provider struct {
	endpoint string
	apiKey   string
}

type request struct {
	Model    string         `json:"model"`
	Messages []inputMessage `json:"messages"`
}

type inputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type response struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func New(apiKey string, endpoint ...string) *Provider {
	providerEndpoint := "https://api.openai.com/v1/responses"
	fmt.Printf(endpoint[0])
	if len(endpoint) > 0 && endpoint[0] != "" {
		providerEndpoint = endpoint[0]
	}

	return &Provider{
		endpoint: providerEndpoint,
		apiKey:   apiKey,
	}
}

func (p *Provider) Generate(ctx context.Context, modelReq models.Request) (*models.Response, error) {
	instructionParts := []string{}
	if modelReq.SystemPrompt != "" {
		instructionParts = append(instructionParts, modelReq.SystemPrompt)
	}

	instructionParts = append(instructionParts, modelReq.Instructions...)

	input := make([]inputMessage, 0, len(modelReq.History)+1)

	input = append(input, inputMessage{
		Role:    string(models.RoleSystem),
		Content: strings.Join(instructionParts, "\n"),
	})

	for _, msg := range modelReq.History {
		input = append(input, inputMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	input = append(input, inputMessage{
		Role:    string(models.RoleUser),
		Content: modelReq.Input,
	})

	body := request{
		Model:    modelReq.Model,
		Messages: input,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	fmt.Printf("HTTP Response:", resp, "\nError:", err)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("openai request failed: %s: %s", resp.Status, string(respBody))
	}

	var result response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &models.Response{
		Text: result.text(),
	}, nil
}

func (r *response) text() string {
	if len(r.Choices) > 0 {
		return r.Choices[0].Message.Content
	}
	return ""
}
