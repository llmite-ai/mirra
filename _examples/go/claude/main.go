package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("http://localhost:4567"), // This is where we're configuring mirra
	)

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.F(anthropic.ModelClaude3_5Sonnet20241022),
		MaxTokens: anthropic.F(int64(1024)),
		Messages: anthropic.F([]anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock("Say hello and a joke")),
		}),
	})
	if err != nil {
		log.Fatalf("failed to create message: %v", err)
	}

	if len(resp.Content) == 0 {
		fmt.Println("No response from Claude.")
		return
	}

	for _, block := range resp.Content {
		if block.Type == anthropic.ContentBlockTypeText {
			fmt.Println(block.Text)
		}
	}
}
