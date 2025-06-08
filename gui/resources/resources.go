package resources

import (
	"fmt"
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func RegisterResourceFromData(gresourceData []byte) error {
	gresource, err := gio.NewResourceFromData(glib.NewBytes(gresourceData))
	if err != nil {
		return fmt.Errorf("Could not load gresources: %w", err)
	}
	gio.ResourcesRegister(gresource)

	return nil
}

func GetBuilderForPath(path string) *gtk.Builder {
	fullPath := filepath.Join("/net/taukakao/BrowserGlue/generated", path)
	return gtk.NewBuilderFromResource(fullPath)
}
