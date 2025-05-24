package server

import (
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

func RunServerBackground(configFile config.NativeConfigFile, extensionName string, exitChan chan error) {
	go func() { exitChan <- RunServer(configFile, extensionName) }()
}

var serverGroup sync.WaitGroup

var stopServers = make(chan bool)

func RunServer(configFile config.NativeConfigFile, extensionName string) error {
	defer logs.Debug("server exited", extensionName)

	serverGroup.Add(1)
	defer serverGroup.Done()

	socketPath := util.GenerateSocketPath(extensionName)

	os.MkdirAll(filepath.Dir(socketPath), 0o775)
	os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)

	if err != nil {
		logs.Error(err)
		return err
	}
	defer listener.Close()

	logs.Info("Server for", extensionName, "listening on", socketPath)

	retries := 5

	stopConnectionSignal := make(chan bool)
	var connectionWait sync.WaitGroup

	for {
		connChan := make(chan net.Conn, 1)
		errChan := make(chan error, 1)
		go acceptConnection(listener, errChan, connChan)
		select {
		case conn := <-connChan:
			retries = 5
			go handleConnection(configFile.Content.Executable, configFile.Path, extensionName, conn, stopConnectionSignal, &connectionWait)

		case err := <-errChan:
			if retries > 0 {
				logs.Warn("retrying connection:", err)
				retries--
				continue
			} else {
				logs.Error("failed to establish connection", err)
				return err
			}

		case <-stopServers:
			logs.Info("closing server for", extensionName)
			for {
				select {
				case stopConnectionSignal <- true:
					continue
				default:
					logs.Debug("Waiting for all connections to exit", extensionName)
					connectionWait.Wait()
					logs.Debug("all connections exited", extensionName)
					return nil
				}
			}
		}
	}
}

func StopServers() {
	for {
		select {
		case stopServers <- true:
			continue
		default:
			logs.Debug("Waiting for all servers to exit")
			serverGroup.Wait()
			logs.Debug("all servers exited")
			return
		}
	}
}
