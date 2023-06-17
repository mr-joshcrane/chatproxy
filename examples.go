package chatproxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Ask sends a question to the GPT-4 API, aiming to receive a relevant and informed answer.
// It facilitates user interaction with GPT-4 for knowledge retrieval or problem-solving.
func Ask(args []string) int {
	if len(args) == 1 {
		fmt.Fprintln(os.Stderr, "must ask a question")
		return 1
	}
	client, err := NewChatGPTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	question := strings.Join(args[1:], " ")
	answer, err := client.Ask(question)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintln(os.Stdout, answer)
	return 0

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

// Chat function initiates the chat with the user and
// enables interaction between user and the chat proxy.
// It orchestrates the entire conversational experience
// with the purpose of assisting the user in various tasks.
func Chat() error {
	client, err := NewChatGPTClient(WithStreaming(true))
	if err != nil {
		return err
	}
	client.Chat()
	return nil
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

// TLDR generates a concise summary of content from a file or URL, aiming to condense important information.
// It utilizes GPT-4 to help users quickly grasp the key points of large texts.
func TLDR(path string) (summary string, err error) {
	client, err := NewChatGPTClient()
	if err != nil {
		return "", err
	}
	return client.TLDR(path)
}
