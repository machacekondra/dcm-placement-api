package v1alpha1

import (
	"context"

	providerapi "github.com/dcm-project/dcm-placement-api/api/v1alpha1/provider"
	"github.com/dcm-project/dcm-placement-api/internal/api/server"
	"github.com/dcm-project/dcm-placement-api/internal/service"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
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
func (s *ServiceHandler) GetHealth(ctx context.Context, request server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	status := "healthy"
	path := "/health"
	return server.GetHealth200JSONResponse{
		Status: &status,
		Path:   &path,
	}, nil
}

// OPTIONS /catalogitems
func (s *ServiceHandler) OptionsCatalogItems(ctx context.Context, request server.OptionsCatalogItemsRequestObject) (server.OptionsCatalogItemsResponseObject, error) {
	return server.OptionsCatalogItems200JSONResponse{
		AllowedMethods: &[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}, nil
}

// (GET /catalogitems)
func (s *ServiceHandler) ListCatalogItems(ctx context.Context, request server.ListCatalogItemsRequestObject) (server.ListCatalogItemsResponseObject, error) {
	items, err := s.store.CatalogItem().List(ctx)
	if err != nil {
		return server.ListCatalogItems400JSONResponse{Error: err.Error()}, err
	}
	return server.ListCatalogItems200JSONResponse{CatalogItems: toCatalogItemList(items)}, nil
}

// (POST /catalogitems)
func (s *ServiceHandler) CreateCatalogItem(ctx context.Context, request server.CreateCatalogItemRequestObject) (server.CreateCatalogItemResponseObject, error) {
	resource, err := s.store.Resource().Get(ctx, string(request.Body.ResourceType))
	if err != nil {
		return server.CreateCatalogItem400JSONResponse{Error: err.Error()}, err
	}

	item, err := s.store.CatalogItem().Create(ctx, &model.CatalogItem{
		ID:         uuid.New(),
		Name:       request.Body.Name,
		Resource:   *resource,
		Parameters: model.JSONContent(request.Body.Parameters),
	})
	if err != nil {
		return server.CreateCatalogItem400JSONResponse{Error: err.Error()}, err
	}

	return server.CreateCatalogItem201JSONResponse(toCatalogItemResponse(*item)), nil
}

// (DELETE /catalogitems/{id})
func (s *ServiceHandler) DeleteCatalogItem(ctx context.Context, request server.DeleteCatalogItemRequestObject) (server.DeleteCatalogItemResponseObject, error) {
	err := s.store.CatalogItem().Delete(ctx, request.Id)
	if err != nil {
		return server.DeleteCatalogItem400JSONResponse{Error: err.Error()}, err
	}
	return server.DeleteCatalogItem204JSONResponse{}, nil
}

// GET /cataloginstances
func (s *ServiceHandler) ListCatalogInstances(ctx context.Context, request server.ListCatalogInstancesRequestObject) (server.ListCatalogInstancesResponseObject, error) {
	instances, err := s.store.CatalogInstance().List(ctx)
	if err != nil {
		return server.ListCatalogInstances400JSONResponse{Error: err.Error()}, err
	}
	return server.ListCatalogInstances200JSONResponse{Instances: toCatalogInstanceList(instances)}, nil
}

// POST /cataloginstances
func (s *ServiceHandler) CreateCatalogInstance(ctx context.Context, request server.CreateCatalogInstanceRequestObject) (server.CreateCatalogInstanceResponseObject, error) {
	// FIXME
	item, err := s.store.CatalogItem().GetByName(ctx, request.Body.CatalogItem.Name)
	if err != nil {
		return server.CreateCatalogInstance400JSONResponse{Error: err.Error()}, err
	}

	placement, err := s.ps.CreateCatalogInstance(ctx, item)
	if err != nil {
		return server.CreateCatalogInstance400JSONResponse{Error: err.Error()}, err
	}

	instance, err := s.store.CatalogInstance().Create(ctx, &model.CatalogInstance{
		Name:          request.Body.Name,
		CatalogItemID: item.ID,
		ID:            uuid.New(),
		ProviderID:    placement.Provider.ID,
		Status:        "placed", // FIXME: use actual status
		InstanceId:    placement.InstanceResponse.Uuid,
	})
	if err != nil {
		return server.CreateCatalogInstance400JSONResponse{Error: err.Error()}, err
	}

	return server.CreateCatalogInstance201JSONResponse(toCatalogInstanceResponse(*instance)), nil
}

// DELETE /cataloginstances/{id}
func (s *ServiceHandler) DeleteCatalogInstance(ctx context.Context, request server.DeleteCatalogInstanceRequestObject) (server.DeleteCatalogInstanceResponseObject, error) {

	instance, err := s.store.CatalogInstance().Get(ctx, request.Id)
	if err != nil {
		return server.DeleteCatalogInstance400JSONResponse{Error: err.Error()}, nil
	}

	err = s.ps.DeleteCatalogInstance(ctx, instance)
	if err != nil {
		return server.DeleteCatalogInstance400JSONResponse{Error: err.Error()}, nil
	}

	err = s.store.CatalogInstance().Delete(ctx, request.Id)
	if err != nil {
		return server.DeleteCatalogInstance400JSONResponse{Error: err.Error()}, nil
	}
	return server.DeleteCatalogInstance204JSONResponse{}, nil
}

func toCatalogItemList(items model.CatalogItemList) []server.CatalogItem {
	catalogItems := []server.CatalogItem{}
	for _, item := range items {
		catalogItems = append(catalogItems, toCatalogItem(item))
	}
	return catalogItems
}

func toCatalogItem(item model.CatalogItem) server.CatalogItem {
	params := map[string]interface{}(item.Parameters)
	return server.CatalogItem{
		Name:         item.Name,
		ResourceType: server.CatalogItemResourceType(item.Resource.ResourceType),
		Parameters:   params,
	}
}

func toCatalogItemResponse(item model.CatalogItem) server.CatalogItemResponse {
	resourceType := server.CatalogItemResponseResourceType(item.Resource.ResourceType)
	return server.CatalogItemResponse{
		Name:         item.Name,
		Parameters:   (map[string]interface{})(item.Parameters),
		ResourceType: resourceType,
		Id:           item.ID,
	}
}

func toCatalogInstanceList(instances model.CatalogInstanceList) []server.CatalogInstanceResponse {
	catalogInstances := []server.CatalogInstanceResponse{}
	for _, instance := range instances {
		catalogInstances = append(catalogInstances, toCatalogInstanceResponse(instance))
	}
	return catalogInstances
}

func toCatalogInstanceResponse(instance model.CatalogInstance) server.CatalogInstanceResponse {
	providerResp := providerapi.ProviderServiceResponse{
		Id:       instance.Provider.ID,
		Name:     instance.Provider.Name,
		Endpoint: instance.Provider.Endpoint,
	}
	return server.CatalogInstanceResponse{
		Id:          &instance.ID,
		Name:        instance.Name,
		CatalogItem: toCatalogItem(instance.CatalogItem),
		Provider:    &providerResp,
		Status:      &instance.Status,
	}
}
