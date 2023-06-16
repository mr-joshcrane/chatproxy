package chatproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
)

type ChatMessage struct {
	Content string
	Role    string
}

const (
	RoleUser   = "user"
	RoleBot    = "assistant"
	RoleSystem = "system"
)

type ChatGPTClient struct {
	client        *openai.Client
	chatHistory   []ChatMessage
	input         io.Reader
	output        io.Writer
	errorStream   io.Writer
	auditTrail    io.Writer
	fixedResponse string
	streaming     bool
}
type ClientOption func(*ChatGPTClient) *ChatGPTClient

func WithOutput(output, err io.Writer) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.output = output
		c.errorStream = err
		return c
	}
}

func WithAudit(audit io.Writer) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.auditTrail = audit
		return c
	}
}

func WithInput(input io.Reader) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.input = input
		return c
	}
}

func WithFixedResponse(response string) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.fixedResponse = response
		return c
	}
}

func WithStreaming(streaming bool) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.streaming = streaming
		return c
	}
}

func NewChatGPTClient(opts ...ClientOption) (*ChatGPTClient, error) {
	token, ok := os.LookupEnv("OPENAI_TOKEN")
	if !ok {
		return nil, errors.New("must have OPENAI_TOKEN env var set")
	}
	file, err := CreateAuditLog()
	if err != nil {
		return nil, err
	}
	c := &ChatGPTClient{
		client:      openai.NewClient(token),
		chatHistory: []ChatMessage{},
		auditTrail:  file,
		input:       os.Stdin,
		output:      os.Stdout,
		errorStream: os.Stderr,
		streaming:   false,
	}
	for _, opt := range opts {
		c = opt(c)
	}
	return c, nil
}

type CompletionOption func(*openai.ChatCompletionRequest) *openai.ChatCompletionRequest

func WithFixedResponseAPIValidate(response string) CompletionOption {
	return func(req *openai.ChatCompletionRequest) *openai.ChatCompletionRequest {
		req.MaxTokens = 1
		req.Stop = []string{response}
		return req
	}
}

func (c *ChatGPTClient) SetPurpose(prompt string) {
	purpose := "PURPOSE: " + prompt
	m := ChatMessage{
		Content: purpose,
		Role:    RoleSystem,
	}
	if len(c.chatHistory) > 0 {
		c.chatHistory[0] = m
	} else {
		c.chatHistory = append(c.chatHistory, m)
	}
	c.Log(RoleSystem, purpose)
}

func (c *ChatGPTClient) GetCompletion(opts ...CompletionOption) (string, error) {
	if c.fixedResponse != "" {
		return c.fixedResponse, nil
	}
	messages := make([]openai.ChatCompletionMessage, len(c.chatHistory))
	for i, message := range c.chatHistory {
		messages[i] = openai.ChatCompletionMessage{
			Content: message.Content,
			Role:    message.Role,
		}
	}
	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4,
		Messages: messages,
		Stream:   true,
	}
	for _, opt := range opts {
		opt(&req)
	}

	stream, err := c.client.CreateChatCompletionStream(context.Background(), req)
	if err != nil {
		err, ok := err.(*openai.APIError)
		if ok {
			if err.HTTPStatusCode == 400 {
				c.LogErr(err)
				c.RollbackLastMessage()
				return fmt.Sprintf("Backing out of transaction: %s", err.Message), nil
			}
			if err.HTTPStatusCode == 401 {
				c.LogErr(err)
				return "", errors.New("unauthorized. Please check your OPENAI_TOKEN env var")
			}
		}
		return "", err
	}
	defer stream.Close()

	discardStreamResp := req.Stop != nil && len(req.Stop) > 0
	if discardStreamResp {
		return req.Stop[0], nil
	}
	if c.streaming {
		return streamedResponse(c, stream)
	}
	return bufferedResponse(stream)
}

func (c *ChatGPTClient) RecordMessage(role string, message string) {
	m := ChatMessage{
		Content: message,
		Role:    role,
	}
	c.chatHistory = append(c.chatHistory, m)
	c.Log(role, message)
}

func (c *ChatGPTClient) RollbackLastMessage() []ChatMessage {
	if len(c.chatHistory) > 1 {
		c.chatHistory = c.chatHistory[:len(c.chatHistory)-1]
	}
	c.Log(RoleSystem, "Last message rolled back")
	return c.chatHistory
}

func streamedResponse(c *ChatGPTClient, stream *openai.ChatCompletionStream) (message string, err error) {
	color.New(color.FgGreen).Fprint(c.output, "ASSISTANT) ")
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			color.New(color.FgGreen).Fprintln(c.output)
			return message, nil
		}

		if err != nil {
			return "", err
		}
		token := response.Choices[0].Delta.Content
		message += token

		color.New(color.FgGreen).Fprint(c.output, token)
	}
}

func bufferedResponse(stream *openai.ChatCompletionStream) (message string, err error) {
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return message, nil
		}

		if err != nil {
			return "", err
		}
		token := response.Choices[0].Delta.Content
		message += token
	}
}
