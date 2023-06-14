# Chatproxy

This README ghostwritten by the `Chat` CLI tool backed by ChatGPT4

Chatproxy is a powerful Golang library that simplifies interactions with OpenAI's GPT-4 model, allowing developers to seamlessly integrate GPT-4 into their Go applications for various tasks. It comes with a collection of ready-to-use command-line tools that serve as examples of how to leverage the chatproxy library. Users can customize the API client, output formats, and authentication methods, making it an indispensable tool for Golang enthusiasts working with AI-powered document generation and processing.

## Key Features

- Effortless integration with GPT-4 for your Go applications
- Simple API client with customizable settings
- Common functions for handling messages, conversation history, and errors
- Collection of handy command-line tools: Ask, Card, Commit, Chat and TLDR

Unlock the power of GPT-4 in your Go projects with Chatproxy and take your applications to the next level.

## Chatproxy Library

### Installation and Usage
```bash
go get -u github.com/mr-joshcrane/chatproxy
```

```go
package main

import (
	"fmt"
	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	answer, err := chatproxy.Ask("What is the capital of France?")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Answer:", answer)
}
```

```bash
Answer: Paris
```

## Ask CLI Tool
### Installation and Usage
```bash
go install github.com/mr-joshcrane/chatproxy/cmd/ask@latest
ask "What is the capital of France?"
The capital of France is Paris.
```

## Cards CLI Tool
### Installation and Usage
```bash
go install github.com/mr-joshcrane/chatproxy/cmd/cards@latest
cards path/to/your/file.txt
Card 1: Example sentence from file.txt.
Card 2: Another example from file.txt.
```

## Commit CLI Tool
### Installation and Usage
```bash
go install github.com/mr-joshcrane/chatproxy/cmd/commit@latest
commit
Accept Generated Message? (Y)es/(N)o
Add installation and usage instructions for Chatproxy library and CLI tools
Generated commit message: "Implement new feature for xyz"
```

## Chat CLI Tool

### Installation and Usage
```bash
go install github.com/mr-joshcrane/chatproxy/cmd/chat@latest
chat
```
```
    SYSTEM) Please describe the purpose of this assistant.
    USER) You help me write Golang documentation.
    USER) >chatproxy
    USER) What special commands are available in the Chat CLI tool and what do they
    do?
    USER) What special commands are availble in the chat CLI tool and what do they do?
    ASSISTANT) In the chat CLI tool, there are THREE special commands:
        1. `>filename.txt`: This command is used to load the contents of a local file into the chat session.
        The text in the file will be used as input for the assistant, and it will base its response on that content.
        To use this command, simply type `>` followed by the name of the file you want to load. For example:

   >file.txt

        2. `<filename.txt`: This command is used to write the assistant's response to a local file.
        You can use this command to save the generated content for future reference.
        To use this command, type `<` followed by the name of the file you want to save the response to,
        and then enter your input prompt. For example:

   <output.txt What is the capital of France?

        3.  The `?` command in the chat CLI tool is used to generate comprehension questions based on the provided text.
        It uses Bloom's Taxonomy (2001) to create questions that help assess the understanding of the given content.
        To use this command, simply type `?` at the beginning of the chat input
        and questions will be generated from the content of the current
        conversation. To make sure you were really paying attention!

These special commands help users extend the interactivity between the chat CLI tool and external files, making it more convenient to use different sources of information or store assistant responses for later use.

```
## TLDR CLI Tool
### Installation and Usage
```bash
go install github.com/mr-joshcrane/chatproxy/cmd/tldr@latest
tldr path/to/your/file.txt
A brief summary of your file.

tldr https://example.site.com
A brief summary of your website.
```
