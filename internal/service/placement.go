package service

import (
	"context"
	"fmt"

	"github.com/dcm-project/dcm-placement-api/internal/api/server"
	"github.com/dcm-project/dcm-placement-api/internal/catalog"
	"github.com/dcm-project/dcm-placement-api/internal/deploy"
	"github.com/dcm-project/dcm-placement-api/internal/handlers/v1alpha1/mappers"
	"github.com/dcm-project/dcm-placement-api/internal/opa"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PlacementService struct {
	store            store.Store
	opa              *opa.Validator
	deploy           *deploy.DeployService
	containerService *deploy.ContainerService
}

func NewPlacementService(store store.Store, opa *opa.Validator,
	deploy *deploy.DeployService, containerService *deploy.ContainerService) *PlacementService {
	return &PlacementService{store: store, opa: opa, deploy: deploy, containerService: containerService}
}

func (s *PlacementService) CreateApplication(ctx context.Context, request *server.CreateApplicationJSONRequestBody, appID string) (*server.ApplicationResponse, error) {
	logger := zap.S().Named("placement_service:create_app")

	// OPA validation:
	tier := 2
	if request.Tier != nil {
		tier = *request.Tier
	}
	logger.Info("Evaluating policy: ", "Tier: ", fmt.Sprintf("%d", tier))
	result, err := s.opa.EvalTierPolicy(ctx, tier, request.Name, request.Zones)
	if err != nil {
		return nil, err
	}

	logger.Info("OPA validation result: ", "Result: ", result)

	if !s.opa.IsValid(result) {
		failures := s.opa.GetFailures(result)
		if len(failures) > 0 {
			return nil, fmt.Errorf("validation failed: %v", failures)
		}
		return nil, fmt.Errorf("input validation failed")
	}

	serviceType := request.Service
	var applicationID uuid.UUID
	if appID != "" {
		applicationID, _ = uuid.Parse(appID)
	} else {
		applicationID = uuid.New()
	}

	// Store in database post validation
	zones := s.opa.GetRequiredZones(result)
	appModel := model.Application{
		ID:      applicationID,
		Name:    request.Name,
		Service: string(request.Service),
		Zones:   zones,
		Tier:    tier,
	}

	app, err := s.store.Application().Create(ctx, appModel)
	if err != nil {
		return nil, err
	}

	// Deploy VMs
	for _, zone := range zones {
		logger.Info("Created service in Zone: ", "Zone: ", zone)
		if serviceType == "webserver" {
			vm := catalog.GetCatalogVm(request.Service)
			err = s.deploy.DeployVM(ctx, request.Name, vm, zone, app.ID.String())
			if err != nil {
				return nil, err
			}
		}
		// Deploy container application
		if serviceType == "container" {
			containerApp := catalog.GetContainerApp()
			err = s.containerService.HandleContainerDeployment(ctx, containerApp, request.Name, zone, app.ID.String())
			if err != nil {
				return nil, err
			}
		}
	}

	appService := string(request.Service)
	return &server.ApplicationResponse{
		Name:    &request.Name,
		Service: &appService,
		Tier:    &tier,
		Id:      &app.ID,
	}, nil
}

func (s *PlacementService) DeleteApplication(ctx context.Context, id uuid.UUID) (*server.ApplicationResponse, error) {
	logger := zap.S().Named("placement_service:delete_app")
	app, err := s.store.Application().Get(ctx, id)
	if err != nil {
		return nil, err
	}

	// Delete VMs from zones
	for _, zone := range app.Zones {
		logger.Info("Removing service from Zone: ", "Zone: ", zone)
		err = s.deploy.DeleteVM(ctx, id.String(), zone)
		if err != nil {
			return nil, err
		}
	}

	// Delete app from database
	err = s.store.Application().Delete(ctx, id)
	if err != nil {
		return nil, err
	}

	return mappers.ApplicationToAPI(*app), nil
}
