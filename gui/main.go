package main

import (
	_ "embed"

	"github.com/taukakao/browser-glue/gui/application"
	"github.com/taukakao/browser-glue/gui/resources"
)

//go:generate ./compile_resources.sh

//go:embed generated/resource.gresource
var gresourceData []byte

func main() {
	resources.RegisterResourceFromData(gresourceData)

	application.RunApplication()
}
