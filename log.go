package chatproxy

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// Log logs a chat message with the given role and message. It helps maintain a comprehensive log of interactions
// in the ChatGPTClient. The primary purpose of this function is to clearly show which role (e.g., user, bot, system)
// is responsible for a particular message in the conversation.
func (c *ChatGPTClient) Log(role string, message string) {
	m := ChatMessage{
		Content: message,
		Role:    role,
	}
	c.logWithFormatting(m)
}

func (c *ChatGPTClient) logWithFormatting(m ChatMessage) {
	formatted := fmt.Sprintf("%s) %s", strings.ToUpper(m.Role), m.Content)
	switch m.Role {
	case RoleBot:
		fmt.Fprintln(c.transcript, formatted)
	case RoleUser:
		fmt.Fprintln(c.transcript, formatted)
	case RoleSystem:
		fmt.Fprintln(c.transcript, formatted)
	default:
		fmt.Fprintln(c.output, formatted) // Default output with no color
	}
}

// LogOut logs a message to the ChatGPTClient's output stream. This is useful for logging messages that are not
// part of the conversation, such as instructions or system status updates.
func (c *ChatGPTClient) LogOut(message ...any) {
	fmt.Fprintln(c.output, message...)
	fmt.Fprintln(c.transcript, message...)
}

// LogErr logs errors in the ChatGPTClient's errorStream.
// This makes it possible to capture and handle errors in a standardized manner, enabling efficient debugging and error handling.
func (c *ChatGPTClient) LogErr(err error) {
	fmt.Fprint(c.errorStream, err)
}

// Prompt formats and prints system prompts to the output. It uses yellow color to differentiate system messages
// from user and bot messages for better visibility. The main purpose of this function is to guide user interactions
// and ensure clear communication of instructions or system status updates.
func (c *ChatGPTClient) Prompt(prompts ...string) {
	for _, prompt := range prompts {
		formattedPrompt := fmt.Sprintf("SYSTEM) %s", prompt)
		color.New(color.FgYellow).Fprintln(c.output, formattedPrompt) // Yellow for system
	}
	fmt.Fprint(c.output, "USER) ")
}

func (c *ChatGPTClient) Fprint(a ...interface{}) {
	fmt.Fprint(c.output, a...)
}
