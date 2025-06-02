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

			err := reloadEnabledServers(browser, listenIn, exitChan)
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

	serverMapKey := createServerMapKey(serv.ConfigFile.Name(), serv.ExtensionName)
	runningServers.Add(serverMapKey, serv)
	defer runningServers.Remove(serverMapKey)

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
	currentlyRunning := runningServers.GetList()
	for _, server := range currentlyRunning {
		server.StopBackground()
	}
	logs.Debug("Waiting for all servers to exit")
	for runningServers.Len() > 0 {
	}
	logs.Debug("all servers exited")
}

var runningServers threadSafeServerMap

func reloadEnabledServers(browser settings.Browser, listenIn bool, exitChan chan error) error {
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

	enabledConfigMap := make(map[string]config.NativeConfigFile)
	for _, config := range nativeConfigs {
		for _, extensionName := range config.Content.AllowedExtensions {
			serverMapKey := createServerMapKey(config.Name(), extensionName)
			enabledConfigMap[serverMapKey] = config
		}
	}

	for _, runningServerKey := range runningServers.GetKeyList() {
		_, enabled := enabledConfigMap[runningServerKey]
		if enabled {
			continue
		}

		server, ok := runningServers.Get(runningServerKey)
		if !ok {
			continue
		}

		server.StopBackground()
	}

	for enabledServerKey, config := range enabledConfigMap {
		_, running := runningServers.Get(enabledServerKey)
		if running {
			continue
		}

		extensionName := extensionNameFromMapKey(enabledServerKey)

		server := Server{ConfigFile: config, ExtensionName: extensionName}

		server.RunBackground(listenIn, exitChan)
	}

	return nil
}

func createServerMapKey(configName string, extensionName string) string {
	serverMapKey := append([]byte(extensionName), 0x7F)
	serverMapKey = append(serverMapKey, []byte(configName)...)
	return string(serverMapKey)
}

func extensionNameFromMapKey(mapKey string) string {
	mapKeyBytes := []byte(mapKey)
	i := slices.Index(mapKeyBytes, 0x7F)
	if i < 0 {
		panic("Map key " + fmt.Sprint(mapKeyBytes) + " with string representation " + mapKey + " is malformed!")
	}
	return string(mapKeyBytes[0:i])
}

type threadSafeServerMap struct {
	servers map[string]*Server
	sync.Mutex
}

func (sl *threadSafeServerMap) Add(key string, s *Server) {
	sl.Lock()
	defer sl.Unlock()

	if sl.servers == nil {
		sl.servers = make(map[string]*Server)
	}

	sl.servers[key] = s
}

func (sl *threadSafeServerMap) Remove(key string) bool {
	sl.Lock()
	defer sl.Unlock()

	if _, ok := sl.servers[key]; !ok {
		return false
	}

	delete(sl.servers, key)
	return true
}

func (sl *threadSafeServerMap) Get(key string) (*Server, bool) {
	sl.Lock()
	defer sl.Unlock()

	server, ok := sl.servers[key]

	return server, ok
}

func (sl *threadSafeServerMap) GetKeyList() []string {
	sl.Lock()
	defer sl.Unlock()

	allKeys := []string{}

	for k := range sl.servers {
		allKeys = append(allKeys, k)
	}

	return allKeys
}

func (sl *threadSafeServerMap) GetList() []*Server {
	sl.Lock()
	defer sl.Unlock()

	allServers := []*Server{}

	for _, v := range sl.servers {
		allServers = append(allServers, v)
	}

	return allServers
}

func (sl *threadSafeServerMap) Len() int {
	return len(sl.servers)
}

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
	go permissionRoutine()
}
