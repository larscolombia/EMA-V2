package openai

import (
	"context"
	"os"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	api         *openai.Client
	AssistantID string
}

func NewClient() *Client {
	key := os.Getenv("OPENAI_API_KEY")
	assistant := os.Getenv("CHAT_PRINCIPAL_ASSISTANT")
	c := openai.NewClient(key)
	return &Client{api: c, AssistantID: assistant}
}

func (c *Client) StreamMessage(ctx context.Context, prompt string) (<-chan string, error) {
	stream, err := c.api.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model: c.AssistantID,
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
