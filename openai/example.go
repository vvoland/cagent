package openai

import (
	"context"
	"fmt"
	"io"

	"github.com/sashabaranov/go-openai"
)

// Example function demonstrating how to use the chat completion stream
// This function processes each chunk of the response as it arrives
func ExampleStreamChatCompletion(client *Client, prompt string, w io.Writer) error {
	ctx := context.Background()

	// Create a messages array with a single user message
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	// Create a streaming chat completion
	stream, err := client.CreateChatCompletionStream(ctx, messages, nil)
	if err != nil {
		return fmt.Errorf("error creating chat completion stream: %w", err)
	}
	defer stream.Close()

	// Process each chunk as it arrives
	for {
		response, err := stream.Recv()
		if err == io.EOF {
			// Stream finished
			break
		}
		if err != nil {
			return fmt.Errorf("error receiving from stream: %w", err)
		}

		// Print each chunk to the provided writer
		fmt.Fprint(w, response.Choices[0].Delta.Content)
	}

	return nil
}

// Example of how to use the standard non-streaming completion
func ExampleChatCompletion(client *Client, prompt string) (string, error) {
	ctx := context.Background()

	// Create a messages array with a single user message
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: prompt,
		},
	}

	// Create a chat completion
	response, err := client.CreateChatCompletion(ctx, messages, nil)
	if err != nil {
		return "", fmt.Errorf("error creating chat completion: %w", err)
	}

	// Return the response content
	return response.Choices[0].Message.Content, nil
}
