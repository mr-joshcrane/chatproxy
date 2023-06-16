package chatproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Ask sends a question to the GPT-4 API, aiming to receive a relevant and informed answer.
// It facilitates user interaction with GPT-4 for knowledge retrieval or problem-solving.
func Ask(question string) (answer string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return "", err
	}
	return client.Ask(question)

}

// Ask sends a user question to the GPT-4 API, and expects an informed response.
// This method is part of the ChatGPTClient and allows users to leverage the GPT-4 model for answering queries.
func (c *ChatGPTClient) Ask(question string) (answer string, err error) {
	c.SetPurpose("Please answer the following question as best you can.")
	c.RecordMessage(RoleUser, question)
	return c.GetCompletion()
}

// Card generates a set of flashcards from a given file or URL, aiming to enhance learning by summarizing important concepts.
// It uses GPT-4 for extracting key information in a compact and easy-to-review format.
func Card(path string) (cards []string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return nil, err
	}
	return client.Card(path)
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

// TLDR generates a concise summary of content from a file or URL, aiming to condense important information.
// It utilizes GPT-4 to help users quickly grasp the key points of large texts.
func TLDR(path string) (summary string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return "", err
	}
	return client.TLDR(path)
}

// TLDR generates a brief summary of the content from a file or URL.
// This method is part of the ChatGPTClient and leverages the GPT-4 API to present an abstract of the main text, providing a quick overview.
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

// Commit analyzes staged Git files, parsing the diff, and generates a meaningful commit message.
// It aims to streamline the process of creating accurate and informative commit descriptions for better version control.
func Commit() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err := cmd.Run()
	if err != nil {
		return errors.New("must be in a git repository")
	}
	client, err := NewChatGPTClient()
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
	if err != nil {
		return err
	}
	r := strings.ToUpper(string(char))
	if r != "Y" {
		return errors.New("generated commit message not accepted")
	}
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	return cmd.Run()
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
