package server

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/taukakao/browser-glue/lib/logs"
)

//go:generate go build -o generated/client-executable ../../client/client.go
//go:embed generated/client-executable
var clientExecutableData []byte

type alreadyCreatedListSafe struct {
	sync.Mutex
	list []string
}

var alreadyCreated alreadyCreatedListSafe

func writeClientExecutable(clientExecutablePath string) error {
	var err error

	alreadyCreated.Lock()
	if slices.Contains(alreadyCreated.list, clientExecutablePath) {
		alreadyCreated.Unlock()
		return nil
	}
	alreadyCreated.list = append(alreadyCreated.list, clientExecutablePath)
	alreadyCreated.Unlock()

	err = os.MkdirAll(filepath.Dir(clientExecutablePath), 0o755)
	if err != nil {
		err = fmt.Errorf("can't create directory for client executable: %w", err)
		logs.Error(err)
		return err
	}

	file, err := os.Create(clientExecutablePath)
	if err != nil {
		err = fmt.Errorf("can't create client executable file: %w", err)
		logs.Error(err)
		return err
	}
	defer file.Close()

	err = file.Chmod(0o755)
	if err != nil {
		err = fmt.Errorf("can't change permissions for client executable file: %w", err)
		logs.Error(err)
		return err
	}

	_, err = file.Write(clientExecutableData)
	if err != nil {
		err = fmt.Errorf("can't write client executable: %w", err)
		logs.Error(err)
		return err
	}

	logs.Info("client executable created in:", clientExecutablePath)

	return nil
}
