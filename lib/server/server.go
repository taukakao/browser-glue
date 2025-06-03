package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

var ErrAlreadyRunning = errors.New("server already running")

type Server struct {
	ConfigFile    config.NativeConfigFile
	ExtensionName string
	ListenIn      bool

	running bool
	stop    chan struct{}
}

func (serv *Server) RunBackground() {
	startServerQueue <- serv
}

func (serv *Server) StopBackground() {
	stopServerQueue <- serv
}

func (serv *Server) end() {
	if !serv.running {
		return
	}
	serv.stop <- struct{}{}
}

func (serv *Server) run() error {
	if serv.running {
		return ErrAlreadyRunning
	}
	serv.running = true

	serv.stop = make(chan struct{}, 1)

	defer logs.Debug("server exited", serv.ExtensionName)

	browser := serv.ConfigFile.GetBrowser()
	writeClientExecutable(browser.GetClientPath())

	socketDir := browser.GetFlatpakRuntimeAppFolder()
	socketFileName := util.GenerateSocketFileName(serv.ExtensionName)

	socketPath := filepath.Join(socketDir, socketFileName)

	os.MkdirAll(filepath.Dir(socketPath), 0o775)
	listener, err := net.Listen("unix", socketPath)

	if err != nil {
		err = fmt.Errorf("can't listen on socket %s: %w", socketPath, err)
		logs.Error(err)
		return err
	}
	defer listener.Close()

	logs.Info("Server for", serv.ExtensionName, "listening on", socketPath)

	retries := 0

	stopConnectionSignal := make(chan bool)
	var connectionWait sync.WaitGroup

	for {
		connChan := make(chan net.Conn, 1)
		errChan := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				errChan <- err
			}
			connChan <- conn
		}()
		select {
		case conn := <-connChan:
			retries = 0
			go handleConnection(serv.ConfigFile.Content.Executable, serv.ConfigFile.Path, serv.ExtensionName, serv.ListenIn, conn, stopConnectionSignal, &connectionWait)

		case err := <-errChan:
			if retries < 5 {
				logs.Warn("retrying connection for", serv.ExtensionName, err)
				retries++
				continue
			} else {
				err = fmt.Errorf("failed to establish connection for %s: %w", serv.ExtensionName, err)
				logs.Error(err)
				return err
			}

		case <-serv.stop:
			logs.Info("closing server for", serv.ExtensionName)
			for {
				select {
				case stopConnectionSignal <- true:
					continue
				default:
					logs.Debug("Waiting for all connections to exit", serv.ExtensionName)
					connectionWait.Wait()
					logs.Debug("all connections exited", serv.ExtensionName)
					return nil
				}
			}
		}
	}
}
