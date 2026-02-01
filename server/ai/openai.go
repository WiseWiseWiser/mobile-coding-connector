package ai

import (
	"context"
	"errors"
	"fmt"
	"io"

	openaisdk "github.com/sashabaranov/go-openai"
)

// getClient creates an OpenAI client configured for the specified provider
func getClient(cfg Config) *openaisdk.Client {
	clientCfg := openaisdk.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}
	return openaisdk.NewClientWithConfig(clientCfg)
}

// CallCompletion calls the AI API for a non-streaming completion
func CallCompletion(ctx context.Context, cfg Config, messages []Message) (string, error) {
	client := getClient(cfg)

	// Convert messages to OpenAI format
	openaiMessages := make([]openaisdk.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openaisdk.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	req := openaisdk.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
	}
	if cfg.MaxTokens > 0 {
		req.MaxTokens = cfg.MaxTokens
	}

	resp, err := client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("AI API error (model: %s): %w", model, err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from AI")
	}

	return resp.Choices[0].Message.Content, nil
}

// CallStream calls the AI API with streaming enabled using the official SDK
func CallStream(ctx context.Context, cfg Config, messages []Message, callback StreamCallback) error {
	client := getClient(cfg)

	// Convert messages to OpenAI format
	openaiMessages := make([]openaisdk.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openaiMessages[i] = openaisdk.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	streamReq := openaisdk.ChatCompletionRequest{
		Model:    model,
		Messages: openaiMessages,
		Stream:   true,
	}
	if cfg.MaxTokens > 0 {
		streamReq.MaxTokens = cfg.MaxTokens
	}

	fmt.Printf("[AI] Creating stream for model: %s\n", model)
	
	stream, err := client.CreateChatCompletionStream(ctx, streamReq)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}
	defer stream.Close()
	fmt.Printf("[AI] Stream created, waiting for responses...\n")

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Printf("[AI] Stream EOF\n")
			callback(StreamChunk{Type: ChunkTypeDone, Content: ""})
			return nil
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Printf("[AI] Stream canceled by client\n")
				return fmt.Errorf("request canceled by client")
			}
			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Printf("[AI] Stream timed out\n")
				return fmt.Errorf("stream timed out")
			}
			fmt.Printf("[AI] Stream error: %v\n", err)
			return fmt.Errorf("stream error: %w", err)
		}

		if len(response.Choices) == 0 {
			continue
		}

		choice := response.Choices[0]

		if choice.FinishReason == openaisdk.FinishReasonStop {
			fmt.Printf("[AI] Stream finished (stop reason)\n")
			callback(StreamChunk{Type: ChunkTypeDone, Content: ""})
			return nil
		}

		// Handle reasoning/thinking content
		reasoningContent := choice.Delta.ReasoningContent
		if reasoningContent != "" {
			// fmt.Printf("[AI] Stream thinking: %s\n", reasoningContent)
			if err := callback(StreamChunk{Type: ChunkTypeThinking, Content: reasoningContent}); err != nil {
				return err
			}
		}

		// Handle normal content
		content := choice.Delta.Content
		if content != "" {
			// fmt.Printf("[AI] Stream content: %s\n", content)
			if err := callback(StreamChunk{Type: ChunkTypeContent, Content: content}); err != nil {
				return err
			}
		}
	}
}
