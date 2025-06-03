package server

import (
	"fmt"
	"slices"
	"sync"

	"github.com/taukakao/browser-glue/lib/config"
	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/settings"
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
		signal.receivers = [](chan<- struct{}){}
	}()
}

var allExitedSignal allExitSignalSafe

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

func init() {
	go serverManagerRoutine()
}
