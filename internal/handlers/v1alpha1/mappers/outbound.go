package mappers

import (
	"github.com/dcm-project/dcm-placement-api/internal/api/server"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
)

func ApplicationToAPI(dbApp model.Application) *server.ApplicationResponse {
	zones := []string(dbApp.Zones)
	return &server.ApplicationResponse{
		Name:    &dbApp.Name,
		Service: &dbApp.Service,
		Tier:    &dbApp.Tier,
		Zones:   &zones,
		Id:      &dbApp.ID,
	}
}

func ApplicationListToAPI(dbApps model.ApplicationList) server.ApplicationList {
	var apiApps []server.ApplicationResponse
	for _, dbApp := range dbApps {
		apiApps = append(apiApps, *ApplicationToAPI(dbApp))
	}
	return server.ApplicationList{Applications: apiApps}
}
