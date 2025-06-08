package main

import (
	_ "embed"

	"github.com/taukakao/browser-glue/gui/application"
)

//go:generate ./compile_resources.sh

//go:embed generated/resource.gresource
var gresourceData []byte

func main() {
	application.RunApplication(gresourceData)
}
