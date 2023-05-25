package chatproxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	client      *openai.Client
	chatHistory []ChatMessage
	input       io.Reader
	output      io.Writer
	errorStream io.Writer
	auditTrail  io.Writer
}

func (c *ChatGPTClient) Log(role string, message string) {
	m := ChatMessage{
		Content: message,
		Role:    role,
	}
	c.logWithFormatting(m)
}

func (c *ChatGPTClient) logWithFormatting(m ChatMessage) {
	formatted := fmt.Sprintf("%s) %s", strings.ToUpper(m.Role), m.Content)
	switch m.Role {
	case RoleBot:
		color.New(color.FgGreen).Fprintln(c.output, formatted) // Green for assistant
	case RoleUser:
		fmt.Fprintln(c.auditTrail, formatted)
	case RoleSystem:
		color.New(color.FgYellow).Fprintln(c.output, formatted) // Yellow for system
	default:
		fmt.Fprintln(c.output, formatted) // Default output with no color
	}
}

func (c *ChatGPTClient) LogErr(err error) {
	fmt.Fprintln(c.errorStream, err)
}

func (c *ChatGPTClient) Prompt(prompts ...string) {
	for _, prompt := range prompts {
		formattedPrompt := fmt.Sprintf("SYSTEM) %s", prompt)
		color.New(color.FgYellow).Fprintln(c.output, formattedPrompt) // Yellow for system
	}
	fmt.Fprint(c.output, "USER) ")
}

type ClientOption func(*ChatGPTClient) *ChatGPTClient

func WithOutput(output, err io.Writer) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.output = output
		c.errorStream = err
		return c
	}
}

func NewChatGPTClient(token string, opts ...ClientOption) (*ChatGPTClient, error) {
	
	file, err := os.Create("audit.txt")
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
	}
	for _, opt := range(opts) {
		c = opt(c)
	}
	return c, nil
}

func (c *ChatGPTClient) SetPurpose(prompt string) {
	purpose := "SYSTEM PURPOSE: " + prompt
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

type CompletionOption func(*openai.ChatCompletionRequest) *openai.ChatCompletionRequest

func WithFixedResponse(response string) CompletionOption {
	return func(req *openai.ChatCompletionRequest) *openai.ChatCompletionRequest {
		req.MaxTokens = 1
		req.Stop = []string{response}
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
				return fmt.Sprintf("Backing out of transaction: %s", err.Message), nil
			}
		}
		return "", err
	}
	if req.Stop != nil && len(req.Stop) > 0 {
		return req.Stop[0], nil
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("invalid response: %+v", resp)
	}
	choice := resp.Choices[0].Message.Content
	return choice, nil
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

func Start() {
	token := os.Getenv("OPENAPI_TOKEN")
	c, err := NewChatGPTClient(token)
	if err != nil {
		c.LogErr(err)
		os.Exit(1)
	}
	c.Prompt("Please describe the purpose of this assistant.")
	scan := bufio.NewScanner(c.input)

	for scan.Scan() {
		var opts []CompletionOption
		line := scan.Text()
		if len(c.chatHistory) == 0 {
			c.SetPurpose(line)
			c.Prompt()
			continue
		}
		if strings.HasPrefix(line, ">") {
			line, err = c.MessageFromFiles(line[1:])
			if err != nil {
				c.LogErr(err)
				c.Prompt()
				continue
			}
			opts = append(opts, WithFixedResponse("Files receieved!"))
		}
		if strings.HasPrefix(line, "<") {
			path, line, ok := strings.Cut(line[1:], " ")
			if !ok {
				c.LogErr(err)
				c.Prompt()
				continue
			}
			c.RecordMessage(RoleUser, line)
			code, err := c.GetCompletion()
			if err != nil {
				c.LogErr(err)
				c.Prompt()
				continue
			}
			MessageToFile(code, path)
			c.Prompt()
			continue

		}
		c.RecordMessage(RoleUser, line)
		if line == "exit" {
			break
		}
		reply, err := c.GetCompletion(opts...)
		if err != nil {
			c.LogErr(err)
			os.Exit(1)
		}
		c.RecordMessage(RoleBot, reply)
		c.Prompt()
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

func (c *ChatGPTClient) MessageFromFiles(path string) (string, error) {
	message := ""
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
			fmt.Fprintf(c.output, "-> %s\n", path)
			message += m
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return message, nil
}

func MessageToFile(content string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	fmt.Fprintln(file, content)
	return nil
}
