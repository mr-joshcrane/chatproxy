package chatproxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Ask sends a question to the GPT-4 API, aiming to receive a relevant and informed answer.
// It facilitates user interaction with GPT-4 for knowledge retrieval or problem-solving.
func Ask(args []string) int {
	client, err := NewChatGPTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(args) == 1 {
		client.LogErr(fmt.Errorf("must ask a question"))
		return 1
	}
	question := strings.Join(args[1:], " ")
	answer, err := client.Ask(question)
	if err != nil {
		client.LogErr(err)
		return 1
	}
	client.LogOut(answer)
	return 0

}

// Card generates a set of flashcards from a given file or URL, aiming to enhance learning by summarizing important concepts.
// It uses GPT-4 for extracting key information in a compact and easy-to-review format.
func Card(args []string) int {
	client, err := NewChatGPTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(args) == 1 {
		client.LogErr(fmt.Errorf("must ask a question"))
		return 1
	}
	path := strings.Join(args[1:], " ")
	cards, err := client.Card(path)
	if err != nil {
		client.LogErr(err)
		return 1
	}
	client.LogOut(cards)
	return 0
}

// Chat function initiates the chat with the user and
// enables interaction between user and the chat proxy.
// It orchestrates the entire conversational experience
// with the purpose of assisting the user in various tasks.
func Chat() int {
	client, err := NewChatGPTClient(WithStreaming(true))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	client.Chat()
	return 0
}

// Commit analyzes staged Git files, parsing the diff, and generates a meaningful commit message.
// It aims to streamline the process of creating accurate and informative commit descriptions for better version control.
func Commit() int {
	client, err := NewChatGPTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	err = cmd.Run()
	if err != nil {
		client.LogErr(fmt.Errorf("not a git repository"))
		return 1
	}
	commitMsg, err := client.Commit()
	if err != nil {
		client.LogErr(err)
		return 1
	}
	fmt.Fprintln(client.output, "Accept Generated Message? (Y)es/(N)o \n"+commitMsg)
	input := bufio.NewReader(client.input)
	char, _, err := input.ReadRune()
	if err != nil {
		client.LogErr(err)
		return 1
	}
	r := strings.ToUpper(string(char))
	if r != "Y" {
		client.LogOut("Commit rejected")
		return 0
	}
	cmd = exec.Command("git", "commit", "-m", commitMsg)
	err = cmd.Run()
	if err != nil {
		client.LogErr(err)
		return 1
	}
	return 0
}

// TLDR generates a concise summary of content from a file or URL, aiming to condense important information.
// It utilizes GPT-4 to help users quickly grasp the key points of large texts.
func TLDR(args []string) int {
	client, err := NewChatGPTClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if len(args) == 1 {
		client.LogErr(fmt.Errorf("must ask a question"))
		return 1
	}
	path := strings.Join(args[1:], " ")
	summary, err := client.TLDR(path)
	if err != nil {
		client.LogErr(err)
		return 1
	}
	client.LogOut(summary)
	return 0
}
