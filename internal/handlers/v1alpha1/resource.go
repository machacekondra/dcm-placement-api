package v1alpha1

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	server "github.com/dcm-project/dcm-placement-api/internal/api/server/catalog"
	"github.com/dcm-project/dcm-placement-api/internal/store"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"gopkg.in/yaml.v2"
)

type ResourceHandler struct {
	store store.Store
}

func NewResourceHandler(store store.Store) *ResourceHandler {
	return &ResourceHandler{
		store: store,
	}
}

// (GET /health)
func (s *ResourceHandler) GetHealth(ctx context.Context, request server.GetHealthRequestObject) (server.GetHealthResponseObject, error) {
	status := "healthy"
	path := "/health"
	return server.GetHealth200JSONResponse{
		Status: &status,
		Path:   &path,
	}, nil
}

// (OPTIONS /resources)
func (s *ResourceHandler) OptionsResources(ctx context.Context, request server.OptionsResourcesRequestObject) (server.OptionsResourcesResponseObject, error) {
	return server.OptionsResources200JSONResponse{
		AllowedMethods: &[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	}, nil
}

// (GET /resources)
func (s *ResourceHandler) ListResources(ctx context.Context, request server.ListResourcesRequestObject) (server.ListResourcesResponseObject, error) {
	resources, err := s.store.Resource().List(ctx)
	if err != nil {
		return server.ListResources400JSONResponse{Error: err.Error()}, err
	}

	resourceList := toResourceList(resources)

	// Check Accept header from context
	if acceptsYAML(ctx) {
		yamlData, err := yaml.Marshal(server.ResourceList{Resources: resourceList})
		if err != nil {
			return server.ListResources400JSONResponse{Error: err.Error()}, err
		}
		return server.ListResources200ApplicationyamlResponse{
			Body:          bytes.NewReader(yamlData),
			ContentLength: int64(len(yamlData)),
		}, nil
	}

	return server.ListResources200JSONResponse{Resources: resourceList}, nil
}

// (GET /resources/{resourceType})
func (s *ResourceHandler) GetResource(ctx context.Context, request server.GetResourceRequestObject) (server.GetResourceResponseObject, error) {
	resource, err := s.store.Resource().Get(ctx, string(request.ResourceType))
	if err != nil {
		return server.GetResource400JSONResponse{Error: err.Error()}, err
	}

	resourceData := toResource(*resource)

	// Check Accept header from context
	if acceptsYAML(ctx) {
		yamlData, err := yaml.Marshal(resourceData)
		if err != nil {
			return server.GetResource400JSONResponse{Error: err.Error()}, err
		}
		return server.GetResource200ApplicationyamlResponse{
			Body:          bytes.NewReader(yamlData),
			ContentLength: int64(len(yamlData)),
		}, nil
	}

	return server.GetResource200JSONResponse(resourceData), nil
}

// (POST /resources/{resourceType})
func (s *ResourceHandler) RegisterServiceProvider(ctx context.Context, request server.RegisterServiceProviderRequestObject) (server.RegisterServiceProviderResponseObject, error) {
	resource, err := s.store.Resource().Get(ctx, string(request.ResourceType))
	if err != nil {
		return server.RegisterServiceProvider400JSONResponse{Error: err.Error()}, err
	}
	provider, err := s.store.ServiceProvider().Create(ctx, &model.ServiceProvider{
		Name:       request.Body.Name,
		Endpoint:   request.Body.Endpoint,
		ResourceID: resource.ID,
		ID:         uuid.New(),
	})
	if err != nil {
		return server.RegisterServiceProvider400JSONResponse{Error: err.Error()}, err
	}
	return server.RegisterServiceProvider201JSONResponse(toProviderService(*provider)), nil
}

func toResourceList(resources model.ResourceList) []server.Resource {
	resourceList := []server.Resource{}
	for _, resource := range resources {
		resourceList = append(resourceList, toResource(resource))
	}
	return resourceList
}

func toResource(resource model.Resource) server.Resource {
	providers := []server.ProviderServiceResponse{}
	for _, provider := range resource.ServiceProviders {
		providers = append(providers, toProviderServiceResponse(provider))
	}

	return server.Resource{
		ResourceType: server.ResourceResourceType(resource.ResourceType),
		Schema:       resource.Content,
		Providers:    &providers,
	}
}

func toProviderService(provider model.ServiceProvider) server.ProviderServiceResponse {
	return server.ProviderServiceResponse{
		Id:       provider.ID,
		Name:     provider.Name,
		Endpoint: provider.Endpoint,
	}
}

func toProviderServiceResponse(provider model.ServiceProvider) server.ProviderServiceResponse {
	return server.ProviderServiceResponse{
		Id:       provider.ID,
		Name:     provider.Name,
		Endpoint: provider.Endpoint,
	}
}

// acceptsYAML checks if the request accepts YAML format
func acceptsYAML(ctx context.Context) bool {
	// Get HTTP request from context
	if req := ctx.Value("http_request"); req != nil {
		if httpReq, ok := req.(*http.Request); ok {
			accept := httpReq.Header.Get("Accept")
			if strings.Contains(accept, "application/yaml") ||
				strings.Contains(accept, "application/x-yaml") ||
				strings.Contains(accept, "text/yaml") {
				return true
			}
		}
	}
	return false
}
