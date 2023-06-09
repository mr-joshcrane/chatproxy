package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "must ask a question")
		os.Exit(1)
	}
	answer, err := chatproxy.Card(strings.Join(os.Args[1:], " "))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(answer)
}
