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
	auditTrail  io.Writer
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
	}, nil
}

func (c *ChatGPTClient) SetPurpose(prompt string) {
	c.RecordMessage(ChatMessage{
		Content: prompt,
		Role:    RoleSystem,
	})
}

func (c *ChatGPTClient) GetCompletion() (string, error) {
	messages := make([]openai.ChatCompletionMessage, len(c.chatHistory))
	for i, message := range c.chatHistory {
		messages[i] = openai.ChatCompletionMessage{
			Content: message.Content,
			Role:    message.Role,
		}
	}

	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4,
			Messages: messages,
		},
	)
	if err != nil {
		err, ok := err.(*openai.APIError); if ok {
			if err.HTTPStatusCode == 400 {
				fmt.Fprintln(os.Stderr, err)
				c.RollbackLastMessage()
				return "Message rolled back out of context.", nil
			}
		}
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("invalid response: %+v", resp)
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *ChatGPTClient) RecordMessage(message ChatMessage) {
	c.chatHistory = append(c.chatHistory, message)
	if c.auditTrail != nil {
		fmt.Fprintln(c.auditTrail, message)
	}
}

func (c *ChatGPTClient) RollbackLastMessage() []ChatMessage {
	if len(c.chatHistory) > 1 {
		c.chatHistory = c.chatHistory[:len(c.chatHistory) -1]
	}
	if c.auditTrail != nil {
		fmt.Fprintln(c.auditTrail, "Context Window Exceeded, rolling back.") 
	}
	return c.chatHistory
}

func Start() {
	token := os.Getenv("OPENAPI_TOKEN")
	chatGPT, err := NewChatGPTClient(token)
	if err != nil {
		panic(err)
	}
	fmt.Fprintln(os.Stdout, "What is my purpose?")
	scan := bufio.NewScanner(os.Stdin)

	for scan.Scan() {
		line := scan.Text()
		if len(chatGPT.chatHistory) == 0 {
			chatGPT.SetPurpose(line)
			continue
		}
		message := ChatMessage{
			Content: line,
			Role:    RoleUser,
		}

		if strings.HasPrefix(line, ">") {
			files, err := MessagesFromFiles(line[1:])
			if err != nil {
				panic(err)
			}
			for _, file := range files {
				chatGPT.RecordMessage(file)
			}
			continue
		}
		chatGPT.RecordMessage(message)
		if line == "exit" {
			break
		}
		reply, err := chatGPT.GetCompletion()
		if err != nil {
			panic(err)
		}
		message = ChatMessage{
			Content: reply,
			Role:    RoleBot,
		}
		chatGPT.RecordMessage(message)
		fmt.Println(message)
	}
}

func MessageFromFile(path string) (ChatMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return ChatMessage{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	content := ""
	for scanner.Scan() {
		content += scanner.Text()
	}
	return ChatMessage{
		Content: fmt.Sprintf("--%s--\n%s", path, content),
		Role:    RoleUser,
	}, nil
}

func MessagesFromFiles(path string) ([]ChatMessage, error) {
	var messages []ChatMessage
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
			messages = append(messages, m)
		}

		return nil
	})
	if err != nil {
		return []ChatMessage{}, err
	}
	return messages, nil
}

func MessageToFile(message ChatMessage, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err 
	}
	fmt.Fprintln(file, message.Content)
	return nil
}

