package gotique

import (
	"context"

	"github.com/google/uuid"
	"github.com/zraisan/gotique/models"
)

const defaultNumHistoryMessages = 10

type Role = models.Role

const (
	RoleSystem    = models.RoleSystem
	RoleUser      = models.RoleUser
	RoleAssistant = models.RoleAssistant
)

type Agent struct {
	Name               string
	Instructions       []string
	SystemPrompt       string
	Model              models.Model
	FallbackModels     []models.Model
	NumHistoryMessages int
	DefaultSessionID   uuid.UUID
	Sessions           map[uuid.UUID]*Session
}

type Session struct {
	SessionID uuid.UUID
	Messages  []Message
}

type Prompt struct {
	Role    Role
	Content string
}

type Message struct {
	prompts [2]Prompt
}

func (m Message) Prompts() [2]Prompt {
	return m.prompts
}

func NewSession() uuid.UUID {
	return uuid.New()
}

func NewAgent(a Agent) *Agent {
	if a.NumHistoryMessages == 0 {
		a.NumHistoryMessages = defaultNumHistoryMessages
	}

	session := &Session{SessionID: NewSession()}

	a.Sessions = map[uuid.UUID]*Session{session.SessionID: session}
	a.DefaultSessionID = session.SessionID

	return &a
}

func (a *Agent) Session(id uuid.UUID) *Session {
	if a.Sessions == nil {
		a.Sessions = make(map[uuid.UUID]*Session)
	}

	if id == uuid.Nil {
		if a.DefaultSessionID == uuid.Nil {
			a.DefaultSessionID = NewSession()
		}

		id = a.DefaultSessionID
	}

	session, ok := a.Sessions[id]
	if !ok {
		session = &Session{SessionID: id}
		a.Sessions[id] = session
	}

	return session
}

func (s *Session) history(n int) []models.Message {
	if n <= 0 || len(s.Messages) == 0 {
		return nil
	}

	window := s.Messages[max(0, len(s.Messages)-n):]

	flat := make([]models.Message, 0, len(window)*2)
	for _, message := range window {
		for _, prompt := range message.prompts {
			flat = append(flat, models.Message{
				Role:    prompt.Role,
				Content: prompt.Content,
			})
		}
	}

	return flat
}

func (a *Agent) Run(ctx context.Context, prompt string, sessionID uuid.UUID) (string, error) {
	session := a.Session(sessionID)

	req := models.Request{
		SystemPrompt: a.SystemPrompt,
		Instructions: a.Instructions,
		History:      session.history(a.NumHistoryMessages),
		Input:        prompt,
	}

	resp, err := a.Model.Generate(ctx, req)
	if err != nil {
		for i := range a.FallbackModels {
			if ctx.Err() != nil {
				break
			}

			resp, err = a.FallbackModels[i].Generate(ctx, req)
			if err == nil {
				break
			}
		}
	}

	if err != nil {
		return "", err
	}

	session.Messages = append(session.Messages, Message{
		prompts: [2]Prompt{
			{Role: RoleUser, Content: prompt},
			{Role: RoleAssistant, Content: resp.Text},
		},
	})

	return resp.Text, nil
}
