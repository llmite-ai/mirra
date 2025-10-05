package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(
		ctx,
		option.WithAPIKey(apiKey),
		option.WithEndpoint("http://localhost:4567"),
	)
	if err != nil {
		log.Fatalf("failed to create generative AI client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-2.5-pro")
	prompt := "Say hello and a joke"
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("failed to generate content: %v", err)
	}

	if len(resp.Candidates) == 0 {
		fmt.Println("No response from Gemini.")
		return
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			fmt.Println(string(text))
			return
		}
	}

	fmt.Println("No text response from Gemini.")
}
