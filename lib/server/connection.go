package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pterm/pterm"
	"github.com/taukakao/browser-glue/lib/logs"
)

func acceptConnection(listener net.Listener, errChan chan error, connChan chan net.Conn) {
	conn, err := listener.Accept()
	if err != nil {
		errChan <- err
	}

	connChan <- conn
}

func handleConnection(commandPath string, configPath string, extensionName string, conn net.Conn, stop chan bool, wg *sync.WaitGroup) error {
	defer logs.Debug("connection exited", extensionName)

	wg.Add(1)
	defer wg.Done()

	var copyWait sync.WaitGroup
	defer copyWait.Wait()

	var err error
	defer conn.Close()

	logs.Info("new connection for", extensionName)

	cmd := exec.Command(commandPath, configPath, extensionName)
	defer cmd.Wait()

	cmd.Dir = filepath.Dir(commandPath)

	stdin, err := cmd.StdinPipe()
	defer stdin.Close()
	if err != nil {
		err = fmt.Errorf("could not open the Stdin pipe for %s: %w", extensionName, err)
		logs.Error(err)
		return err
	}

	stdout, err := cmd.StdoutPipe()
	defer stdout.Close()
	if err != nil {
		err = fmt.Errorf("could not open the Stdout pipe for %s: %w", extensionName, err)
		logs.Error(err)
		return err
	}

	err = cmd.Start()
	if err != nil {
		err = fmt.Errorf("could not start the command for %s: %w", extensionName, err)
		logs.Error(err)
		return err
	}

	exitChan := make(chan error, 2)

	go customCopyGo(stdin, conn, &copyWait, exitChan, extensionName, true)
	go customCopyGo(conn, stdout, &copyWait, exitChan, extensionName, false)

	select {
	case <-stop:
	case err := <-exitChan:
		if err == nil {
			break
		}
		if errors.Is(err, io.EOF) {
			logs.Debug("end of stream", extensionName)
			break
		}
		err = fmt.Errorf("failed to copy stream for %s: %w", extensionName, err)
		logs.Error(err)
	}

	logs.Info("stopping connection for", extensionName)

	return nil
}

func customCopyGo(dst io.Writer, src io.Reader, wg *sync.WaitGroup, exitChan chan error, extensionName string, isreceiver bool) {
	wg.Add(1)
	defer wg.Done()
	var err error

	_, err = io.Copy(io.MultiWriter(dst, &sniffer{extensionName, isreceiver}), src)
	exitChan <- err
}

type sniffer struct {
	string
	bool
}

func (s *sniffer) Write(p []byte) (retlen int, reterr error) {
	retlen = len(p)
	reterr = nil

	for len(p) >= 4 {
		msgSizeEncoded := p[:4]
		p = p[4:]

		messageSize := uint32(msgSizeEncoded[3]) << 8 * 3
		messageSize |= uint32(msgSizeEncoded[2]) << 8 * 2
		messageSize |= uint32(msgSizeEncoded[1]) << 8
		messageSize |= uint32(msgSizeEncoded[0])

		if uint64(messageSize) > uint64(len(p)) {
			err := errors.New("message too large to decode: " + string(p))
			logs.Error(err)
			return
		}

		message := p[:messageSize]
		p = p[messageSize:]

		var result map[string]any

		err := json.Unmarshal(message, &result)
		if err != nil {
			err := errors.New("can't unmarshal message: " + string(message))
			logs.Error(err)
			return
		}

		parts := []string{}
		for key, value := range result {
			parts = append(parts, key+": "+fmt.Sprint(value))
		}

		final := strings.Join(parts, "\n") + "\n"

		if s.bool {
			pterm.NewRGB(200, 100, 0).Println(final)
		} else {
			pterm.NewRGB(0, 100, 100).Println(final)
		}
	}
	return
}
