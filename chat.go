package chatproxy

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Chat method handles the conversational flow for
// the ChatGPTClient, aiming to provide a seamless
// user experience by managing prompts and strategies.
func (c *ChatGPTClient) Chat() {
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

const (
	QuestionPrompt = `Given the above text, generate some reading comprehension questions.
	If I respond to the questions, you will give me a score out of 10 and how I can improve my answer.
	Use Bloom's Taxonomy (2001) to generate the questions. Do not generate questions about Bloom's Taxonomy.
	Produce only the questions, the user will provide the answers.
	
	BOT: Q: What is the end goal of teaching.
	USER: A: To know the answers to questions.
	BOT: Feedback: 2/10 - This demonstrates only a surface understanding. 
	USER: A: To transfer knowledge in such a way that the learner can apply it in new situations.
	BOT: Feedback: 10/10 - This gets a the heart of the answer.
	`
)

type Strategy interface {
	Execute(*ChatGPTClient) error
}

type FileLoad struct{ input string }

// Execute method for FileLoad strategy handles loading
// file contents to be processed by ChatGPTClient, which
// enables users to provide input via files instead of
// just through the chat interface.
func (s FileLoad) Execute(c *ChatGPTClient) error {
	line, err := c.GetContent(s.input[1:])
	if err != nil {
		c.LogErr(err)
		return err
	}
	c.RecordMessage(RoleUser, line)
	reply, err := c.GetCompletion(WithFixedResponseAPIValidate("Files receieved!"))
	if err != nil {
		c.LogErr(err)
		return err
	}
	c.RecordMessage(RoleBot, reply)
	return nil
}

type FileWrite struct{ input string }

// Execute method for FileWrite strategy allows writing
// output from the chat interaction to a file, offering
// an organized and convenient way to store results.
func (s FileWrite) Execute(c *ChatGPTClient) error {
	path, line, ok := strings.Cut(s.input[1:], " ")
	if !ok {
		return fmt.Errorf("need a file and a prompt to write a file")
	}
	c.RecordMessage(RoleUser, line)
	code, err := c.GetCompletion()
	if err != nil {
		return err
	}
	return MessageToFile(code, path)
}

type Default struct{ input string }

// Execute method for Default strategy is responsible for
// managing a typical chat interaction by sending user input
// to the OpenAI API and receiving a response.
func (s Default) Execute(c *ChatGPTClient) error {
	c.RecordMessage(RoleUser, s.input)
	reply, err := c.GetCompletion()
	if err != nil {
		return err
	}
	c.RecordMessage(RoleBot, reply)
	return nil
}

type Exit struct{}

// Execute method for Exit strategy gracefully manages
// the termination of the chat session when the user
// decides to exit.
func (s Exit) Execute(c *ChatGPTClient) error {
	c.Log(RoleUser, "*exit*")
	return io.EOF
}

// GetStrategy method selects the appropriate strategy
// based on the user input, ensuring the correct action
// is taken to achieve the user's desired outcome.
func (c *ChatGPTClient) GetStrategy(input string) Strategy {
	if strings.HasPrefix(input, ">") {
		return FileLoad{input}
	} else if strings.HasPrefix(input, "<") {
		return FileWrite{input}
	} else if input == "exit" {
		return Exit{}
	} else if strings.HasPrefix(input, "?") {
		return Default{QuestionPrompt}
	} else {
		return Default{input}
	}
}
