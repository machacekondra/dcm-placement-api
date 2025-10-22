package catalog

import (
	"github.com/dcm-project/dcm-placement-api/internal/api/server"
)

type CatalogVm struct {
	Ram int
	Cpu int
	Os  string
}

func GetCatalogVm(serviceName server.ApplicationService) *CatalogVm {
	if serviceName == "webserver" {
		return &CatalogVm{
			Ram: 1,
			Cpu: 1,
			Os:  "fedora",
		}
	}

	return nil
}

type ContainerApp struct {
	Image   string
	Port    int
	Replica int32
}

func GetContainerApp() *ContainerApp {
	return &ContainerApp{
		Image:   "nginx:latest",
		Port:    80,
		Replica: int32(2),
	}
}
