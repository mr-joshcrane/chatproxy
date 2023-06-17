package main

import (
	"os"

	"github.com/mr-joshcrane/chatproxy"
)

func main() {
	os.Exit(chatproxy.TLDR(os.Args))
}
