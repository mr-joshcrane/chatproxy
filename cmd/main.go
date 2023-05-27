package main

import (
	"fmt"
	"os"

	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	token, ok := os.LookupEnv("OPENAPI_TOKEN"); if !ok {
		fmt.Fprintln(os.Stdout, "OPENAPI_TOKEN must be set.")
		os.Exit(1)
	}
	c, err := chatproxy.NewChatGPTClient(token)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c.Start()
}
