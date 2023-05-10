package chatproxy

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
)

type ChatMessage struct {
	Content string
	Role    string
}

const RoleUser = "user"
const RoleBot = "assistant"

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
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("invalid response: %+v", resp)
	}
	return resp.Choices[0].Message.Content, nil
}

func (c *ChatGPTClient) recordMessage(message ChatMessage) {
	c.chatHistory = append(c.chatHistory, message)
	if c.auditTrail != nil {
		fmt.Fprintln(c.auditTrail, message)
	}
}

func Start() {
	token := os.Getenv("OPENAI_API_KEY")
	chatGPT, err := NewChatGPTClient(token)
	if err != nil {
		panic(err)
	}
	scan := bufio.NewScanner(os.Stdin)
	for scan.Scan() {
		line := scan.Text()
		message := ChatMessage{
			Content: line,
			Role:    RoleUser,
		}
		chatGPT.recordMessage(message)

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
		chatGPT.recordMessage(message)
		fmt.Println(message)
	}
}
