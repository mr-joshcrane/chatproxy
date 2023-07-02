package chatproxy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/sashabaranov/go-openai"
)

// ChatMessage represents a message in the chat, providing context and
// a way to model conversation between different participant roles (e.g., user, bot, system).
type ChatMessage struct {
	Content string
	Role    string
}

// Role constants that represent the role of the message sender
const (
	RoleUser   = "user"
	RoleBot    = "assistant"
	RoleSystem = "system"
)

// ChatGPTClient manages interactions with a GPT-based chatbot, providing a way
// to organize the conversation, handle input/output, and maintain an audit trail.
type ChatGPTClient struct {
	client        *openai.Client
	chatHistory   []ChatMessage
	input         io.Reader
	output        io.Writer
	errorStream   io.Writer
	transcript    io.Writer
	fixedResponse string
	streaming     bool
	embeddings    []Embedding
}

type Embedding struct {
	Origin         string
	OriginSequence int
	PlainText      string
	Vector         []float64
}

type Similarities struct {
	Query           string
	RelevantVectors []Similarity
}

type Similarity struct {
	PlainText string
	Score     float64
}

// ClientOption is used to flexibly configure the ChatGPTClient to meet various requirements
// and use cases, such as custom input/output handling or error reporting.
type ClientOption func(*ChatGPTClient) *ChatGPTClient

// WithToken uses the provided token for authentication
// when creating a new ChatGPTClient.
func WithToken(token string) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.client = openai.NewClient(token)
		return c
	}
}

// WithOutput allows customizing the output/error handling in the ChatGPTClient, making the client
// more adaptable to different environments or reporting workflows.
func WithOutput(output, err io.Writer) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.output = output
		c.errorStream = err
		return c
	}
}

// WithTranscript enables keeping a log of all conversation messages, ensuring a persistent record that
// can be useful for auditing, debugging, or further analysis.
func WithTranscript(audit io.Writer) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.transcript = audit
		return c
	}
}

// WithInput assigns a custom input reader for ChatGPTClient, allowing the client to read input
// from any source, offering improved flexibility and adaptability.
func WithInput(input io.Reader) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.input = input
		return c
	}
}

// WithFixedResponse configures the ChatGPTClient to return a predetermined response, offering
// quicker or consistent replies, or simulating specific behavior for test cases.
func WithFixedResponse(response string) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.fixedResponse = response
		return c
	}
}

// WithStreaming controls the streaming mode of the ChatGPTClient, giving the user the choice
// between streamed responses for real-time interactions or buffered responses for complete replies.
func WithStreaming(streaming bool) ClientOption {
	return func(c *ChatGPTClient) *ChatGPTClient {
		c.streaming = streaming
		return c
	}
}

var NewChatGPTClient = DefaultGPTClient

// NewChatGPTClient initializes the ChatGPTClient with the desired options, allowing customization
// through functional options so the client can be tailored to specific needs or requirements.
func DefaultGPTClient(opts ...ClientOption) (*ChatGPTClient, error) {
	file, err := CreateAuditLog()
	if err != nil {
		return nil, err
	}
	c := &ChatGPTClient{
		client:      nil,
		chatHistory: []ChatMessage{},
		transcript:  file,
		input:       os.Stdin,
		output:      os.Stdout,
		errorStream: os.Stderr,
		streaming:   false,
	}
	for _, opt := range opts {
		c = opt(c)
	}
	if c.client == nil {
		token, ok := os.LookupEnv("OPENAI_API_KEY")
		if !ok {
			return nil, errors.New("must have OPENAI_API_KEY env var set or pass token explicitly")
		}
		c.client = openai.NewClient(token)
	}
	return c, nil
}

func (c *ChatGPTClient) TranscriptPath() string {
	if file, ok := c.transcript.(*os.File); ok {
		return file.Name()
	}
	return ""
}

// Ask sends a user question to the GPT-4 API, and expects an informed response.
// This method is part of the ChatGPTClient and allows users to leverage the GPT-4 model for answering queries.
func (c *ChatGPTClient) Ask(question string) (answer string, err error) {
	c.SetPurpose("Please answer the following question as best you can.")
	c.RecordMessage(RoleUser, question)
	return c.GetCompletion()
}

// Card creates flashcards using the content from a given file or URL.
// This method, part of the ChatGPTClient, uses the GPT-4 API to break down and condense information into manageable flashcards.
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
	msg, err := c.GetContent(path)
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

// TLDR generates a brief summary of the content from a file or URL.
// This method is part of the ChatGPTClient and leverages the GPT-4 API to present an abstract of the main text, providing a quick overview.
func (c *ChatGPTClient) TLDR(path string) (summary string, err error) {
	c.SetPurpose("Please summarise the provided text as best you can. The shorter the better.")
	var msg string
	msg, err = c.GetContent(path)
	if err != nil {
		return "", err
	}
	c.RecordMessage(RoleUser, msg)
	return c.GetCompletion()
}

// Commit parses the diff of staged Git files and generates an appropriate commit message.
// This method, part of the ChatGPTClient, helps users maintain clear commit history and conveys changes in a concise and descriptive manner.
func (c *ChatGPTClient) Commit() (summary string, err error) {
	c.SetPurpose(`Please read the git diff provided and write an appropriate commit message.
	Focus on the lines that start with a + (line added) or - (line removed)`)
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

// CompletionOption is used to customize the behavior of the openai.ChatCompletionRequest
// to suit different use cases, such as setting stop words or modifying token limits.
type CompletionOption func(*openai.ChatCompletionRequest) *openai.ChatCompletionRequest

// WithFixedResponseAPIValidate still makes an API call (ensuring request and token length validation) but
// enforces a specific response from the chatbot, ensuring a known output
// and avoiding unpredictable or unnecessary responses during validation.
func WithFixedResponseAPIValidate(response string) CompletionOption {
	return func(req *openai.ChatCompletionRequest) *openai.ChatCompletionRequest {
		req.MaxTokens = 1
		req.Stop = []string{response}
		return req
	}
}

// SetPurpose defines the purpose of the conversation, providing contextual guidance for the chatbot
// to follow, and aligning the conversation towards a specific topic or goal.
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

// GetCompletion retrieves a response from the chatbot based on the conversation history and any
// additional options applied.
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
		var apiErr *openai.APIError
		if errors.As(err, &apiErr) {
			if apiErr.HTTPStatusCode == http.StatusBadRequest {
				c.LogErr(err)
				c.RollbackLastMessage()
				return fmt.Sprintf("Backing out of transaction: %s", apiErr.Message), nil
			}
			if apiErr.HTTPStatusCode == http.StatusUnauthorized {
				c.LogErr(err)
				return "", errors.New("unauthorized. Please check your OPENAI_API_KEY env var or pass a token in explicitly")
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

func (c *ChatGPTClient) CreateEmbeddings(origin string, contents io.Reader) {
	chunks := c.Chunk(contents, 500)
	// Create batches of 500
	var batches [][]string
	for i := 0; i < len(chunks); i += 500 {
		end := i + 500
		if end > len(chunks) {
			end = len(chunks)
		}
		batches = append(batches, chunks[i:end])
	}
	for _, batch := range batches {
		embedding, err := c.Vectorize(origin, batch)
		if err != nil {
			c.LogErr(err)
			continue
		}
		c.embeddings = append(c.embeddings, embedding...)
	}
}

func (c *ChatGPTClient) Chunk(contents io.Reader, chunkSize int) []string {
	var chunks []string
	scanner := bufio.NewScanner(contents)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		chunk := scanner.Text()
		chunk = strings.TrimSpace(chunk)
		if len(chunk) == 0 {
			continue
		}
		chunks = append(chunks, chunk)
	}
	var groupedChunks []string
	if len(chunks) < chunkSize {
		groupedChunks = append(groupedChunks, strings.Join(chunks, " "))
	} else {
		for i := 0; i < len(chunks); i += chunkSize {
			end := i + chunkSize
			if end > len(chunks) {
				end = len(chunks)
			}
			groupedChunks = append(groupedChunks, strings.Join(chunks[i:end], " "))
		}
	}

	return groupedChunks
}

func (c *ChatGPTClient) Vectorize(origin string, s []string) ([]Embedding, error) {
	var embeddings []Embedding
	emb := s
	req := openai.EmbeddingRequest{
		Model: openai.AdaEmbeddingV2,
		Input: emb,
	}
	resp, err := c.client.CreateEmbeddings(context.Background(), req)
	if err != nil {
		return nil, err
	}

	for i, embedding := range resp.Data {
		v := float32ToFloat64(embedding.Embedding)
		embeddings = append(embeddings, Embedding{
			Origin:         origin,
			OriginSequence: i + 1,
			PlainText:      s[i],
			Vector:         v,
		})

	}
	return embeddings, nil
}

func float32ToFloat64(f []float32) []float64 {
	var d []float64
	for _, v := range f {
		d = append(d, float64(v))
	}
	return d
}

func (c *ChatGPTClient) Relevant(query string) (Similarities, error) {
	var similarities Similarities
	similarities.Query = query
	// Vectorize the query
	q, err := c.Vectorize("query", []string{query})
	if err != nil {
		return Similarities{}, err
	}
	for _, v := range c.embeddings {
		similarity := Similarity{
			PlainText: v.PlainText,
			Score:     cosineSimilarity(q[0].Vector, v.Vector),
		}
		similarities.RelevantVectors = append(similarities.RelevantVectors, similarity)
	}
	return similarities, nil
}

func (s Similarities) Top(n int) []string {
	var top []string
	sort.Slice(s.RelevantVectors, func(i, j int) bool {
		return s.RelevantVectors[i].Score > s.RelevantVectors[j].Score
	})
	fmt.Println(len(s.RelevantVectors))
	for i := 0; i < n; i++ {
		top = append(top, s.RelevantVectors[i].PlainText)
	}
	return top
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, magA, magB float64
	for i := range a {
		dot += a[i] * b[i]
		magA += math.Pow(a[i], 2)
		magB += math.Pow(b[i], 2)
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// RecordMessage adds a new message in the conversation context, allowing the chatbot to
// maintain a conversation context. The role parameter provides a mechanism for inserting
// bot or system responses in addition to user messages.
func (c *ChatGPTClient) RecordMessage(role string, message string) {
	m := ChatMessage{
		Content: message,
		Role:    role,
	}
	c.chatHistory = append(c.chatHistory, m)
	c.Log(role, message)
}

// RollbackLastMessage serves as an undo functionality, removing the last message from the conversation,
// and providing a way to recover from erroneous input or chatbot responses.
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
