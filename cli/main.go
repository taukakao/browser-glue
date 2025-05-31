package main

import (
	_ "embed"
	"fmt"
	"os"
)

//go:generate go build -o generated/client-executable ../client/client.go
//go:embed generated/client-executable
var ClientExecutableData []byte

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
