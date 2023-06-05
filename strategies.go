package chatproxy

import (
	"fmt"
	"strings"
)

type Strategy interface {
	Execute(*ChatGPTClient) error
}

type FileLoad struct{ input string }

func (s FileLoad) Execute(c *ChatGPTClient) error {
	line, err := c.MessageFromFiles(s.input[1:])
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

func (s Default) Execute(c *ChatGPTClient) error {
	c.RecordMessage(RoleUser, s.input)
	reply, err := c.GetCompletion()
	if err != nil {
		return err
	}
	c.RecordMessage(RoleBot, reply)
	return nil
}

func (c *ChatGPTClient) GetStrategy(input string) Strategy {
	if strings.HasPrefix(input, ">") {
		return FileLoad{input}
	} else if strings.HasPrefix(input, "<") {
		return FileWrite{input}
	} else {
		return Default{input}
	}

}
