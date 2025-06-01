package server

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
	"github.com/taukakao/browser-glue/lib/util"
)

var ErrNoConfigFiles = errors.New("no config files found")

func RunEnabledServersBackground(browser settings.Browser, listenIn bool, exitChan chan error) error {
	nativeConfigs, err := config.CollectEnabledConfigFiles(browser)
	if err != nil {
		err = fmt.Errorf("can't collect config files: %w", err)
		logs.Error(err)
		return err
	}
	if len(nativeConfigs) == 0 {
		return ErrNoConfigFiles
	}
	for _, nativeConfig := range nativeConfigs {
		for _, extension := range nativeConfig.Content.AllowedExtensions {
			serv := Server{ConfigFile: nativeConfig, ExtensionName: extension}
			serv.RunBackground(listenIn, exitChan)
		}
	}

	changes := make(chan struct{})
	settings.SubscribeToChanges(changes)

	go func() {
		for {
			<-changes

			err := reloadEnabledServers(browser, exitChan)
			if err != nil {
				err = fmt.Errorf("failed reloading servers: %w", err)
				logs.Error(err)

				StopServers()
			}
		}
	}()

	return nil
}

var ErrAlreadyRunning = errors.New("server already running")

type Server struct {
	ConfigFile    config.NativeConfigFile
	ExtensionName string

	running bool
	stop    chan struct{}
}

func (serv *Server) RunBackground(listenIn bool, exitChan chan error) {
	go func() { exitChan <- serv.Run(listenIn) }()
}

func (serv *Server) StopBackground() {
	if !serv.running {
		return
	}
	serv.stop <- struct{}{}
}

func (serv *Server) Run(listenIn bool) error {
	if serv.running {
		return ErrAlreadyRunning
	}
	serv.running = true

	serv.stop = make(chan struct{}, 1)

	runningServers.Add(serv)
	defer runningServers.Remove(serv)

	defer logs.Debug("server exited", serv.ExtensionName)

	socketPath := util.GenerateSocketPath(util.GetClientExecutableDir(), serv.ExtensionName)

	os.MkdirAll(filepath.Dir(socketPath), 0o775)
	os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)

	if err != nil {
		err = fmt.Errorf("can't listen on socket %s: %w", socketPath, err)
		logs.Error(err)
		return err
	}
	defer listener.Close()

	logs.Info("Server for", serv.ExtensionName, "listening on", socketPath)

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
			go handleConnection(serv.ConfigFile.Content.Executable, serv.ConfigFile.Path, serv.ExtensionName, listenIn, conn, stopConnectionSignal, &connectionWait)

		case err := <-errChan:
			if retries > 0 {
				logs.Warn("retrying connection:", err)
				retries--
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

func StopServers() {
	currentlyRunning := runningServers.Copy()
	for _, server := range currentlyRunning {
		server.StopBackground()
	}
	logs.Debug("Waiting for all servers to exit")
	for runningServers.Len() > 0 {
	}
	logs.Debug("all servers exited")
}

var runningServers threadSafeServerList

func reloadEnabledServers(browser settings.Browser, exitChan chan error) error {
	currentlyRunning := runningServers.Copy()

	nativeConfigs, err := config.CollectEnabledConfigFiles(browser)
	if err != nil {
		err = fmt.Errorf("can't collect config files: %w", err)
		logs.Error(err)
		return err
	}
	if len(nativeConfigs) == 0 {
		StopServers()
		return nil
	}

	for _, runningServer := range currentlyRunning {
		enabled := slices.ContainsFunc(nativeConfigs, func(conf config.NativeConfigFile) bool {
			pathMatches := conf.Path == runningServer.ConfigFile.Path
			extensionMatches := slices.Contains(conf.Content.AllowedExtensions, runningServer.ExtensionName)
			return pathMatches && extensionMatches
		})
		if enabled {
			continue
		}

		runningServer.StopBackground()
	}

	for _, nativeConfig := range nativeConfigs {
		for _, extension := range nativeConfig.Content.AllowedExtensions {
			running := slices.ContainsFunc(currentlyRunning, func(serv *Server) bool {
				pathMatches := nativeConfig.Path == serv.ConfigFile.Path
				extensionMatches := extension == serv.ExtensionName
				return pathMatches && extensionMatches
			})
			if running {
				continue
			}

			serv := Server{ConfigFile: nativeConfig, ExtensionName: extension}
			serv.RunBackground(false, exitChan)
		}
	}
	return nil
}

type threadSafeServerList struct {
	servers []*Server
	sync.Mutex
}

func (sl *threadSafeServerList) Add(s *Server) {
	sl.Lock()
	defer sl.Unlock()

	sl.servers = append(sl.servers, s)
}

func (sl *threadSafeServerList) Remove(s *Server) bool {
	sl.Lock()
	defer sl.Unlock()

	index := slices.Index(sl.servers, s)
	if index < 0 {
		return false
	}

	sl.servers = slices.Delete(sl.servers, index, index+1)
	return true
}

func (sl *threadSafeServerList) Copy() []*Server {
	sl.Lock()
	defer sl.Unlock()

	currentlyRunning := make([]*Server, len(sl.servers))
	copy(currentlyRunning, sl.servers)
	return currentlyRunning
}

func (sl *threadSafeServerList) Len() int {
	return len(sl.servers)
}
