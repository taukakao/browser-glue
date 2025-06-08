package resources

import (
	"path/filepath"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

func GetBuilderForPath(path string) *gtk.Builder {
	fullPath := filepath.Join("/net/taukakao/BrowserGlue/generated", path)
	return gtk.NewBuilderFromResource(fullPath)
}
