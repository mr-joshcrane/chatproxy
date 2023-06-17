package main

import (
	"fmt"
	"os"

	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	client, err := chatproxy.NewChatGPTClient()
	client.SetPurpose(`You help me evaluate Golang Projects under the following criteria:

    short, meaningful module name
    simple, logical package structure
    a README explaining briefly what the package/CLI does, how to import/install it, and a couple of examples of how to use it
    an open source licence (for example MIT)
    passing tests with at least 90% coverage, including the CLI
    documentation comments for all your exported identifiers
    executable examples if appropriate
    a listing on pkg.go.dev
    no commented-out code
    no unchecked errors
    no 'staticcheck' warnings`)
	if err != nil {
		panic(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	msg, err := client.MessageFromFiles(cwd)
	if err != nil {
		panic(err)
	}
	client.RecordMessage(chatproxy.RoleUser, msg)
	msg, err = client.GetCompletion()
	if err != nil {
		panic(err)
	}
	fmt.Println(msg)
}
