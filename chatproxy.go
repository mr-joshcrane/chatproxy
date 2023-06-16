package chatproxy

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Ask sends a question to the GPT-4 API and returns a generated answer.
func Ask(question string) (answer string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return "", err
	}
	return client.Ask(question)

}

// Ask takes a user question and returns an answer from the GPT-4 API.
func (c *ChatGPTClient) Ask(question string) (answer string, err error) {
	c.SetPurpose("Please answer the following question as best you can.")
	c.RecordMessage(RoleUser, question)
	return c.GetCompletion()
}

// Card creates a set of flashcards from the content of a file or URL.
func Card(path string) (cards []string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return nil, err
	}
	return client.Card(path)
}

// Card generates a list of flashcards from a given file or URL
// using a ChatContext to process the input.
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

// TLDR generates a concist summary of the content from a file or URL.
func TLDR(path string) (summary string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return "", err
	}
	return client.TLDR(path)
}

// TLDR generates a summary of the content from a file or URL
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

// Commit takes the currently staged files, parses the diff, and generates a commit message.
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

// Commit takes the currently staged files, parses the diff, and generates a commit message.
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
