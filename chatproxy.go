package chatproxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	client      *openai.Client
	chatHistory []ChatMessage
	input       io.Reader
	output      io.Writer
	errorStream io.Writer
	auditTrail  io.Writer
}

func (c *ChatGPTClient) Log(args ...any) {
	fmt.Fprintln(c.output, args...)
	if c.auditTrail != nil {
		fmt.Fprintln(c.auditTrail, args...)
	}
}

func (c *ChatGPTClient) Logf(format string, args ...any) {
	format = format + "\n"
	fmt.Fprintf(c.output, format, args...)
	if c.auditTrail != nil {
		fmt.Fprintf(c.auditTrail, format, args...)
	}
}

func (c *ChatGPTClient) LogErr(err error) {
	fmt.Fprintln(c.errorStream, err)
}

func (c *ChatGPTClient) LogAudit(args ...any) {
	if c.auditTrail != nil {
		fmt.Fprintln(c.auditTrail, args...)
	}
}

func NewChatGPTClient(token string) (*ChatGPTClient, error) {
	file, err := os.Create("audit.txt")
	if err != nil {
		return nil, err
	}
	return &ChatGPTClient{
		client:      openai.NewClient(token),
		chatHistory: []ChatMessage{},
		auditTrail:  file,
		input:       os.Stdin,
		output:      os.Stdout,
		errorStream: os.Stderr,
	}, nil
}

func (c *ChatGPTClient) SetPurpose(prompt string) {
	c.RecordMessage(ChatMessage{
		Content: prompt,
		Role:    RoleSystem,
	})
	c.LogAudit()
}

type CompletionOption func(*openai.ChatCompletionRequest) *openai.ChatCompletionRequest

func WithTokenLimit(tokenLimit int) CompletionOption {
	return func(req *openai.ChatCompletionRequest) *openai.ChatCompletionRequest {
		req.MaxTokens = tokenLimit
		return req
	}
}

func (c *ChatGPTClient) GetCompletion(opts ...CompletionOption) (string, error) {
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
	}
	for _, opt := range opts {
		opt(&req)
	}
	resp, err := c.client.CreateChatCompletion(context.Background(), req)
	if err != nil {
		err, ok := err.(*openai.APIError)
		if ok {
			if err.HTTPStatusCode == 400 {
				c.LogErr(err)
				c.RollbackLastMessage()
				return "Message rolled back out of context.", nil
			}
		}
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("invalid response: %+v", resp)
	}
	choice := resp.Choices[0].Message.Content
	c.Log(choice)
	return choice, nil
}

func (c *ChatGPTClient) RecordMessage(message ChatMessage) {
	c.chatHistory = append(c.chatHistory, message)
	c.LogAudit(message)
}

func (c *ChatGPTClient) RollbackLastMessage() []ChatMessage {
	if len(c.chatHistory) > 1 {
		c.chatHistory = c.chatHistory[:len(c.chatHistory)-1]
	}
	c.Log("Context Window Exceeded, rolling back.")
	return c.chatHistory
}

func Start() {
	token := os.Getenv("OPENAPI_TOKEN")
	c, err := NewChatGPTClient(token)
	if err != nil {
		c.LogErr(err)
		os.Exit(1)
	}
	c.Log("How can I help?")
	scan := bufio.NewScanner(c.input)

	for scan.Scan() {
		var opts []CompletionOption
		line := scan.Text()
		if len(c.chatHistory) == 0 {
			c.SetPurpose(line)
			continue
		}
		message := ChatMessage{
			Content: line,
			Role:    RoleUser,
		}

		if strings.HasPrefix(line, ">") {
			message, err = MessageFromFiles(line[1:])
			if err != nil {
				continue
			}
			opts = append(opts, WithTokenLimit(0))
		}
		c.RecordMessage(message)
		if line == "exit" {
			break
		}
		reply, err := c.GetCompletion(opts...)
		if err != nil {
			c.LogErr(err)
			os.Exit(1)
		}
		message = ChatMessage{
			Content: reply,
			Role:    RoleBot,
		}
		c.RecordMessage(message)
	}
}

func MessageFromFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	content := ""
	for scanner.Scan() {
		content += scanner.Text()
	}

	message := fmt.Sprintf("--%s--\n%s\n", path, content)
	return message, nil
}

func MessageFromFiles(path string) (ChatMessage, error) {
	message := ChatMessage{
		Content: "",
		Role:    RoleUser,
	}
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Ignore hidden files
		if filepath.Base(path)[0] == '.' {
			if info.IsDir() {
				return filepath.SkipDir // If it's a directory, skip it entirely
			}
			return nil // If it's a file, just skip this file
		}

		if !info.IsDir() { // check if it's a file and not a directory
			m, err := MessageFromFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "-> %s\n", path)
			message.Content += m
		}
		return nil
	})
	if err != nil {
		return ChatMessage{}, err
	}
	return message, nil
}

func MessageToFile(message ChatMessage, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	fmt.Fprintln(file, message.Content)
	return nil
}
