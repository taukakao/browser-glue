package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"path/filepath"
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

func handleConnection(commandPath string, configPath string, extensionName string, listenIn bool, conn net.Conn, stop chan bool, wg *sync.WaitGroup) error {
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

	go customCopyGo(stdin, conn, &copyWait, exitChan, extensionName, true, listenIn)
	go customCopyGo(conn, stdout, &copyWait, exitChan, extensionName, false, listenIn)

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

func customCopyGo(dst io.Writer, src io.Reader, wg *sync.WaitGroup, exitChan chan error, extensionName string, isreceiver bool, enableSniffer bool) {
	wg.Add(1)
	defer wg.Done()
	var err error

	if enableSniffer {
		dst = io.MultiWriter(dst, &sniffer{extensionName: extensionName, isReceiver: isreceiver})
	}
	_, err = io.Copy(dst, src)
	exitChan <- err
}

type sniffer struct {
	extensionName string
	isReceiver    bool
	remaining     uint32
	message       []byte
}

const kilobyte = 1000

func (s *sniffer) Write(p []byte) (retlen int, reterr error) {
	retlen = len(p)
	reterr = nil

	for len(p) > 0 {

		if s.remaining == 0 {
			if len(p) < 4 {
				return
			}
			msgSizeEncoded := p[:4]
			p = p[4:]

			messageSize := uint32(msgSizeEncoded[3]) << 8 * 3
			messageSize |= uint32(msgSizeEncoded[2]) << 8 * 2
			messageSize |= uint32(msgSizeEncoded[1]) << 8
			messageSize |= uint32(msgSizeEncoded[0])

			s.remaining = messageSize
		}

		if len(p) < int(s.remaining) {
			s.message = append(s.message, p...)
			s.remaining -= uint32(len(p))
			return
		}

		s.message = append(s.message, p[:s.remaining]...)
		p = p[s.remaining:]
		s.remaining = 0

		s.printout()

		s.message = []byte{}
	}
	return
}

func (s *sniffer) printout() {
	var coloredOutput pterm.RGB
	if s.isReceiver {
		pterm.NewRGB(200, 100, 0).Println(s.extensionName, "-> App")
		coloredOutput = pterm.NewRGB(250, 150, 0)
	} else {
		pterm.NewRGB(0, 150, 150).Println("App ->", s.extensionName)
		coloredOutput = pterm.NewRGB(0, 200, 200)
	}
	defer coloredOutput.Println("")

	var encoded any
	err := json.Unmarshal(s.message, &encoded)
	if err != nil {
		coloredOutput.Println(string(s.message))
		return
	}

	resultEncoded, err := json.MarshalIndent(encoded, "", "   ")
	if err != nil {
		coloredOutput.Println(string(s.message))
		return
	}

	coloredOutput.Println(string(resultEncoded))
}
