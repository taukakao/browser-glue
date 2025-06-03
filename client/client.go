package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/taukakao/browser-glue/lib/util"
)

func printSimpleError(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
}

func main() {
	var err error

	isChrome := len(os.Args) == 2
	isFirefox := len(os.Args) == 3

	var extensionName string
	if isFirefox {
		extensionName = os.Args[2]
	} else if isChrome {
		extensionName = os.Args[1]
	} else {
		printSimpleError("unsupported browser")
		os.Exit(1)
	}

	exePath, err := os.Executable()
	if err != nil {
		printSimpleError("can't determine path of executable:", err)
	}

	socketFileName := util.GenerateSocketFileName(extensionName)

	socketPath := filepath.Join(filepath.Dir(exePath), socketFileName)

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		printSimpleError("failed to dial unix socket", err)
		os.Exit(1)
		return
	}
	defer conn.Close()

	errEvent := make(chan error)

	go copyWithErr(conn, os.Stdin, errEvent)
	go copyWithErr(os.Stdout, conn, errEvent)

	err = <-errEvent
	if err != nil {
		printSimpleError("writing to socket failed", err)
	}
}

func copyWithErr(dst io.Writer, src io.Reader, errEvent chan error) {
	_, err := io.Copy(dst, src)
	errEvent <- err
}
