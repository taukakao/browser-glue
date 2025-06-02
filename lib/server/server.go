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
	"github.com/taukakao/browser-glue/lib/flatpak"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
	"github.com/taukakao/browser-glue/lib/util"
)

func RunEnabledServersBackground(browser settings.Browser, listenIn bool, allServersExited chan<- struct{}) {
	if allServersExited != nil {
		allExitedSignal.subscribe(allServersExited)
	}

	changes := make(chan struct{})
	settings.SubscribeToChanges(changes)

	go func() {
		for {
			<-changes

			err := refreshEnabledServers(browser, listenIn)
			if err != nil {
				err = fmt.Errorf("failed reloading servers: %w", err)
				logs.Error(err)

				StopServers()
			}
		}
	}()

	changes <- struct{}{}
}

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

	checkAndFixPermissionsQueue <- serv.ConfigFile.GetBrowser()

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

func StopServers() {
	allExited := make(chan struct{}, 1)
	allExitedSignal.subscribe(allExited)

	stopAllServers <- struct{}{}

	logs.Debug("Waiting for all servers to exit")
	<-allExited
	logs.Debug("all servers exited")
}

func refreshEnabledServers(browser settings.Browser, listenIn bool) error {
	enabledNativeConfigs, err := config.CollectEnabledConfigFiles(browser)
	if err != nil {
		err = fmt.Errorf("can't collect config files: %w", err)
		logs.Error(err)
		return err
	}
	if len(enabledNativeConfigs) == 0 {
		logs.Warn("No config files are currently enabled.")
		StopServers()
		return nil
	}

	runningServers.Lock()
	defer runningServers.Unlock()

	for _, runningServer := range runningServers.servers {
		if !runningServer.ConfigFile.IsEnabled() {
			runningServer.StopBackground()
		}
	}

	for _, enabledConfig := range enabledNativeConfigs {
		runningExtensions := []string{}
		for _, runningServer := range runningServers.servers {
			if enabledConfig.Matches(&runningServer.ConfigFile) {
				runningExtensions = append(runningExtensions, runningServer.ExtensionName)
			}
		}

		for _, extensionName := range enabledConfig.Content.AllowedExtensions {

			if slices.Contains(runningExtensions, extensionName) {
				continue
			}

			server := Server{ConfigFile: enabledConfig, ExtensionName: extensionName, ListenIn: listenIn}

			server.RunBackground()
		}
	}

	return nil
}

type runningServersSafe struct {
	sync.Mutex
	servers []*Server
}

var runningServers = runningServersSafe{}

var startServerQueue chan *Server = make(chan *Server)
var stopServerQueue chan *Server = make(chan *Server)
var stopAllServers chan struct{} = make(chan struct{})

func serverManagerRoutine() {
	for {
		select {
		case server := <-startServerQueue:
			go func() {
				runningServers.Lock()
				runningServers.servers = append(runningServers.servers, server)
				runningServers.Unlock()

				err := server.run()
				if err != nil {
					logs.Error(err)
				}

				runningServers.Lock()
				runningServers.servers = slices.DeleteFunc(runningServers.servers, func(element *Server) bool { return element == server })
				if len(runningServers.servers) == 0 {
					allExitedSignal.broadcast()
				}
				runningServers.Unlock()
			}()

		case server := <-stopServerQueue:
			server.end()

		case <-stopAllServers:
			runningServers.Lock()
			for _, server := range runningServers.servers {
				go server.end()
			}
			if len(runningServers.servers) == 0 {
				allExitedSignal.broadcast()
			}
			runningServers.Unlock()
		}
	}
}

type allExitSignalSafe struct {
	sync.Mutex
	receivers []chan<- struct{}
}

func (signal *allExitSignalSafe) subscribe(c chan<- struct{}) {
	signal.Lock()
	defer signal.Unlock()
	signal.receivers = append(signal.receivers, c)
}

func (signal *allExitSignalSafe) broadcast() {
	go func() {
		signal.Lock()
		defer signal.Unlock()
		for _, receiver := range signal.receivers {
			receiver <- struct{}{}
		}
	}()
}

var allExitedSignal allExitSignalSafe

var checkAndFixPermissionsQueue chan settings.Browser = make(chan settings.Browser, 10)

func permissionRoutine() {
	for browser := range checkAndFixPermissionsQueue {
		hasPermissions, err := flatpak.CheckBrowserPermissions(browser)
		if err != nil {
			err = fmt.Errorf("could not check browser permissions of %s: %w", browser, err)
			logs.Error(err)
			continue
		}

		if hasPermissions {
			continue
		}

		err = flatpak.FixBrowserPermissions(browser)
		if err != nil {
			err = fmt.Errorf("could not give browser %s the needed flatpak permissions %w", browser, err)
			logs.Error(err)
			continue
		}
	}
}

func init() {
	go serverManagerRoutine()
	go permissionRoutine()
}
