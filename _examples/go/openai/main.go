package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("http://localhost:4567/v1"), // This is where we're configuring mirra
	)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4o,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage("Say hello and a joke"),
		},
	})
	if err != nil {
		log.Fatalf("failed to create chat completion: %v", err)
	}

	if len(resp.Choices) == 0 {
		fmt.Println("No response from OpenAI.")
		return
	}

	for _, choice := range resp.Choices {
		fmt.Println(choice.Message.Content)
	}
}
