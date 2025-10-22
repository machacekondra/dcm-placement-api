package v1alpha1

import (
	"context"

	"github.com/dcm-project/dcm-placement-api/internal/api/server"
	"github.com/dcm-project/dcm-placement-api/internal/handlers/v1alpha1/mappers"
	"github.com/dcm-project/dcm-placement-api/internal/service"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"go.uber.org/zap"
)

type ServiceHandler struct {
	ps    *service.PlacementService
	store store.Store
}

func NewServiceHandler(store store.Store, placementService *service.PlacementService) *ServiceHandler {
	return &ServiceHandler{
		store: store,
		ps:    placementService,
	}
}

// (GET /health)
func (s *ServiceHandler) ListHealth(ctx context.Context, request server.ListHealthRequestObject) (server.ListHealthResponseObject, error) {
	return server.ListHealth200Response{}, nil
}

// (GET /applications)
func (s *ServiceHandler) ListApplications(ctx context.Context, request server.ListApplicationsRequestObject) (server.ListApplicationsResponseObject, error) {
	applications, err := s.store.Application().List(ctx)
	if err != nil {
		return server.ListApplications400JSONResponse{}, nil
	}
	return server.ListApplications200JSONResponse(mappers.ApplicationListToAPI(applications)), nil
}

// (DELETE /applications/{id})
func (s *ServiceHandler) DeleteApplication(ctx context.Context, request server.DeleteApplicationRequestObject) (server.DeleteApplicationResponseObject, error) {
	logger := zap.S().Named("placement_service")
	logger.Info("Deleting Application. ", "Application: ", request)

	app, err := s.ps.DeleteApplication(ctx, request.Id)
	if err != nil {
		logger.Error("Failed to delete Application: ", "error", err)
		return server.DeleteApplication400JSONResponse{}, nil
	}
	logger.Info("Application deleted. ", "Application: ", request.Id)
	return server.DeleteApplication204JSONResponse(*app), nil
}

// (POST /applications)
func (s *ServiceHandler) CreateApplication(ctx context.Context, request server.CreateApplicationRequestObject) (server.CreateApplicationResponseObject, error) {
	logger := zap.S().Named("placement_service")
	logger.Info("Creating Application. ", "Application: ", request)

	paramId := ""
	if (request.Params != server.CreateApplicationParams{}) {
		paramId = *request.Params.Id
	}
	app, err := s.ps.CreateApplication(ctx, request.Body, paramId)
	if err != nil {
		logger.Error("Failed to create Application: ", "error", err)
		return server.CreateApplication400JSONResponse{Error: err.Error()}, nil
	}
	logger.Info("Application created. ", "Application: ", app)
	return server.CreateApplication201JSONResponse(*app), nil
}
