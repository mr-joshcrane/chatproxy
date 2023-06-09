package main

import (
	"fmt"
	"os"
	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	err := chatproxy.Chat()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

