package chatproxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/cixtor/readability"
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
	for _, opt := range opts {
		c = opt(c)
	}
	return c, nil
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

type CompletionOption func(*openai.ChatCompletionRequest) *openai.ChatCompletionRequest

func WithFixedResponseAPIValidate(response string) CompletionOption {
	return func(req *openai.ChatCompletionRequest) *openai.ChatCompletionRequest {
		req.MaxTokens = 1
		req.Stop = []string{response}
		return req
	}
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
		}
		return "", err
	}
	defer stream.Close()

	discardStreamResp := req.Stop != nil && len(req.Stop) > 0
	if discardStreamResp {
		return req.Stop[0], nil
	}
	return StreamResponse(c, stream)
}
func StreamResponse(c *ChatGPTClient, stream *openai.ChatCompletionStream) (message string, err error) {
	// color.New(color.FgGreen).Fprint(c.output, "ASSISTANT) ")
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			// color.New(color.FgGreen).Fprintln(c.output)
			return message, nil
		}

		if err != nil {
			return "", err
		}
		token := response.Choices[0].Delta.Content
		message += token

		// color.New(color.FgGreen).Fprint(c.output, token)
	}
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

func (c *ChatGPTClient) Start() {
	c.Prompt("Please describe the purpose of this assistant.")
	scan := bufio.NewScanner(c.input)

	for scan.Scan() {
		line := scan.Text()
		if len(c.chatHistory) == 0 {
			c.SetPurpose(line)
			c.Prompt()
			continue
		}
		strategy := c.GetStrategy(line)
		err := strategy.Execute(c)
		if err == io.EOF {
			break
		}
		if err != nil {
			c.LogErr(err)
		}
		c.Prompt()
	}
}

func Ask(question string) (answer string, err error) {
	token, ok := os.LookupEnv("OPENAI_TOKEN")
	if !ok {
		return "", errors.New("must have OPENAI_TOKEN env var set")
	}

	client, err := NewChatGPTClient(token)
	if err != nil {
		return "", err
	}
	return client.Ask(question)

}

func (c *ChatGPTClient) Ask(question string) (answer string, err error) {
	c.SetPurpose("Please answer the following question as best you can.")
	c.RecordMessage(RoleUser, question)
	return c.GetCompletion()
}

func Card(path string) (cards []string, err error) {
	token, ok := os.LookupEnv("OPENAI_TOKEN")
	if !ok {
		return nil, errors.New("must have OPENAI_TOKEN env var set")
	}

	client, err := NewChatGPTClient(token)
	if err != nil {
		return nil, err
	}
	return client.Card(path)
}

func (c *ChatGPTClient) Card(path string) (cards []string, err error) {
	c.SetPurpose(`Please generate flashcards from the user provided information.
		Answers should be short.
		A good flashcard look like this:
		---
		Question: What does 'Seperation of Concerns' mean?
		Answer: It means that each function should do one thing and do it well.
		---
		Question: What does 'Liscov Substitution Principle' mean?
		Answer: It means that any class that is the child of another class should be able to be used in place of the parent class.
		---
`)
	msg, err := c.inputOutput(path)
	if err != nil {
		return nil, err
	}
	c.RecordMessage(RoleUser, msg)
	msg, err = c.GetCompletion()
	if err != nil {
		return nil, err
	}
	cards = strings.Split(msg, "---")
	return cards, nil

}

func TLDR(path string) (summary string, err error) {
	token, ok := os.LookupEnv("OPENAI_TOKEN")
	if !ok {
		return "", errors.New("must have OPENAI_TOKEN env var set")
	}
	client, err := NewChatGPTClient(token)
	if err != nil {
		return "", err
	}
	return client.TLDR(path)
}

func (c *ChatGPTClient) inputOutput(path string) (msg string, err error) {
	_, err = os.Stat(path)
	if err == nil {
		msg, err = c.MessageFromFiles(path)
		if err != nil {
			return "", err
		}
	} else {
		_, err  := url.ParseRequestURI(path)
		if err != nil {
			path = "https://" + path
		}
		resp, err := http.Get(path)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		r := readability.New()
		article, err := r.Parse(resp.Body, path)
		if err != nil {
			return "", err
		}
		msg = article.TextContent
	}
	return msg, nil
}

func (c *ChatGPTClient) TLDR(path string) (summary string, err error) {
	c.SetPurpose("Please summarise the provided text as best you can. The shorter the better.")
	var msg string
	msg, err = c.inputOutput(path)
	if err != nil {
		return "", err
	}
	c.RecordMessage(RoleUser, msg)
	return c.GetCompletion()
}

func Commit() error {
	token, ok := os.LookupEnv("OPENAI_TOKEN")
	if !ok {
		return errors.New("must have OPENAI_TOKEN env var set")
	}
	client, err := NewChatGPTClient(token)
	if err != nil {
		return err
	}
	commitMsg, err := client.Commit()
	if err != nil {
		return err
	}
	fmt.Fprintln(client.output, "Accept Generated Message? (Y)es/(N)o \n"+commitMsg)
	input := bufio.NewReader(client.input)
	char, _, err := input.ReadRune()
	r := strings.ToUpper(string(char))
	if r != "Y" {
		return errors.New("generated commit message not accepted")
	}
	cmd := exec.Command("git", "commit", "-m", fmt.Sprintf("%s", commitMsg))
	return cmd.Run()
}

func (c *ChatGPTClient) Commit() (summary string, err error) {
	c.SetPurpose("Please read the git diff provided and write an appropriate commit message.")
	cmd := exec.Command("git", "diff", "--cached")
	buf := bytes.Buffer{}
	cmd.Stdout = &buf
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	if len(buf.String()) == 0 {
		return "", errors.New("no files staged for commit")
	}
	c.RecordMessage(RoleUser, buf.String())
	return c.GetCompletion()
}

func IsValidURL(path string) bool {
	_, err := url.Parse(path)
	if err != nil {
		return false
	}
	return true
}
