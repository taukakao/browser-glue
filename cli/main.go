package main

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/taukakao/browser-glue/cli/commands"
)

//go:generate go build -o generated/client-executable ../client/client.go
//go:embed generated/client-executable
var clientExecutableData []byte

func main() {
	if err := commands.Execute(clientExecutableData); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
