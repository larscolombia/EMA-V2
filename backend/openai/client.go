package openai

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	api         *openai.Client
	AssistantID string
	Model       string
}

func NewClient() *Client {
	key := os.Getenv("OPENAI_API_KEY")
	assistant := os.Getenv("CHAT_PRINCIPAL_ASSISTANT")
	model := os.Getenv("CHAT_MODEL")
	var api *openai.Client
	if key != "" {
		api = openai.NewClient(key)
	}
	return &Client{api: api, AssistantID: assistant, Model: model}
}

func (c *Client) StreamMessage(ctx context.Context, prompt string) (<-chan string, error) {
	// If API or model is not configured, emit a minimal placeholder stream to avoid server 500s
	if c.api == nil || c.AssistantID == "" {
		ch := make(chan string, 1)
		go func() {
			defer close(ch)
			ch <- ""
		}()
		return ch, nil
	}
	// Resolve model to use: prefer CHAT_MODEL; if AssistantID looks like an assistant, fallback to a sane default
	model := c.Model
	if model == "" {
		if len(c.AssistantID) >= 5 && c.AssistantID[:5] == "asst_" {
			// Assistant IDs are not models; default to a general model unless CHAT_MODEL is provided
			model = "gpt-4o-mini"
		} else {
			// If AssistantID is actually a model name, allow using it directly
			model = c.AssistantID
		}
	}

	stream, err := c.api.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	ch := make(chan string)

	go func() {
		defer stream.Close()
		defer close(ch)
		for {
			resp, err := stream.Recv()
			if err != nil {
				break
			}
			if len(resp.Choices) > 0 {
				ch <- resp.Choices[0].Delta.Content
			}
		}
	}()

	return ch, nil
}
