package models

import (
	"context"
	"errors"
)

type Provider interface {
	Generate(ctx context.Context, req Request) (*Response, error)
}

type Model struct {
	name     string
	provider Provider
}

func New(name string, provider Provider) Model {
	return Model{
		name:     name,
		provider: provider,
	}
}

func (m Model) Name() string {
	return m.name
}

func (m Model) Generate(ctx context.Context, req Request) (*Response, error) {
	if m.name == "" {
		return nil, errors.New("model name is required")
	}

	if m.provider == nil {
		return nil, errors.New("model provider is required")
	}

	req.Model = m.name

	return m.provider.Generate(ctx, req)
}

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model        string
	SystemPrompt string
	Instructions []string
	History      []Message
	Input        string
}

type Response struct {
	Text string
}
