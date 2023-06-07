package main

import (
	"fmt"
	"os"
	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, "must ask a question")
		os.Exit(1)
	}
	summary, err := chatproxy.TLDR(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(summary)
}

