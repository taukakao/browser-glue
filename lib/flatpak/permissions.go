package flatpak

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/taukakao/browser-glue/lib/logs"
	"github.com/taukakao/browser-glue/lib/util"
)

func CheckBrowserPermissions(browser util.Browser) (bool, error) {
	var err error

	browserId := browser.GetFlatpakId()

	cmd := exec.Command("flatpak", "--user", "override", "--show", browserId)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		logs.Error(fmt.Sprintf("flatpak command returned:\n%s", stderr.String()))
		err = fmt.Errorf("failed to run flatpak permission override check for %s: %w", browserId, err)
		logs.Error(err)
		return false, err
	}

	clientExecutableDir := util.GetClientExecutableDir()
	path := util.MakePathHomeRelative(clientExecutableDir)

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "filesystems=") {
			continue
		}

		if strings.Contains(line, path) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		err = fmt.Errorf("failed to read output of flatpak permission override command for %s: %w", browserId, err)
		logs.Error(err)
	}

	return false, nil
}

func FixBrowserPermissions(browser util.Browser) error {
	browserId := browser.GetFlatpakId()

	logs.Info("setting browser permissions for", browserId)

	clientExecutableDir := util.GetClientExecutableDir()
	path := util.MakePathHomeRelative(clientExecutableDir)

	cmd := exec.Command("flatpak", "--user", "override", "--filesystem", path, browserId)

	output, err := cmd.CombinedOutput()
	if err != nil {
		logs.Error(fmt.Sprintf("flatpak command returned:\n%s", output))
		err = fmt.Errorf("could not set browser permissions: %w", err)
		logs.Error(err)
		return err
	}

	return nil
}
