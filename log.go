package chatproxy

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

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
		fmt.Fprintln(c.auditTrail, formatted)
	case RoleUser:
		fmt.Fprintln(c.auditTrail, formatted)
	case RoleSystem:
		fmt.Fprintln(c.auditTrail, formatted)
		color.New(color.FgYellow).Fprintln(c.output, formatted) // Yellow for system
	default:
		fmt.Fprintln(c.output, formatted) // Default output with no color
	}
}

func (c *ChatGPTClient) LogErr(err error) {
	fmt.Fprintln(c.errorStream, err)
}

func (c *ChatGPTClient) Prompt(prompts ...string) {
	for _, prompt := range prompts {
		formattedPrompt := fmt.Sprintf("SYSTEM) %s", prompt)
		color.New(color.FgYellow).Fprintln(c.output, formattedPrompt) // Yellow for system
	}
	fmt.Fprint(c.output, "USER) ")
}
