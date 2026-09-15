package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/zraisan/gotique"
	"github.com/zraisan/gotique/models"
	"github.com/zraisan/gotique/providers/openailike"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	provider := openailike.New(os.Getenv("LLM_API_KEY"), os.Getenv("LLM_BASE_URL"))
	model := models.New(os.Getenv("LLM_MODEL"), provider)

	agent := gotique.NewAgent(gotique.Agent{
		Name:               "assistant",
		Model:              model,
		SystemPrompt:       "You are a helpful assistant. You start every response with 'Howdy'",
		Instructions:       []string{"Answer clearly and concisely."},
		NumHistoryMessages: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	chat := uuid.Nil

	ask(ctx, agent, chat, "My name is Abdullah and my favourite language is Go.")
	ask(ctx, agent, chat, "What is my name, and which language do I like?")

	other := gotique.NewSession()
	ask(ctx, agent, other, "What is my name?")

	transcript(agent, chat)
}

func ask(ctx context.Context, agent *gotique.Agent, sessionID uuid.UUID, prompt string) {
	response, err := agent.Run(ctx, prompt, sessionID)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("> %s\n%s\n\n", prompt, response)
}

func transcript(agent *gotique.Agent, sessionID uuid.UUID) {
	session := agent.Session(sessionID)

	fmt.Printf("--- session %s: %d exchanges ---\n", session.SessionID, len(session.Messages))

	for _, message := range session.Messages {
		for _, prompt := range message.Prompts() {
			fmt.Printf("%-9s %s\n", prompt.Role, prompt.Content)
		}
	}
}
