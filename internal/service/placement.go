package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/dcm-project/dcm-placement-api/internal/opa"
	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
)

type InstanceResponse struct {
	Uuid   string `json:"uuid"`
	Status string `json:"status"`
}

type PlacementResponse struct {
	Provider         model.ServiceProvider
	InstanceResponse *InstanceResponse
}

type PlacementService struct {
	opa *opa.Validator
}

func NewPlacementService(opa *opa.Validator) *PlacementService {
	return &PlacementService{opa: opa}
}

func (s *PlacementService) CreateCatalogInstance(ctx context.Context, item *model.CatalogItem) (*PlacementResponse, error) {
	// Very cool placement logic here:
	if len(item.Resource.ServiceProviders) == 0 {
		return nil, fmt.Errorf("no service providers available for this resource")
	}
	provider := item.Resource.ServiceProviders[0]
	if len(item.Resource.ServiceProviders) > 1 {
		// select random provider
		idx := int(uuid.New().ID() % uint32(len(item.Resource.ServiceProviders)))
		provider = item.Resource.ServiceProviders[idx]
	}

	// Prepare JSON body from item.Parameters
	reqBody, err := json.Marshal(item.Parameters)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(provider.Endpoint, "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	instanceResponse := &InstanceResponse{}
	err = json.NewDecoder(resp.Body).Decode(instanceResponse)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to create catalog instance: %s, %s", resp.Status, resp.Body)
	}

	return &PlacementResponse{Provider: provider, InstanceResponse: instanceResponse}, nil
}

func (s *PlacementService) DeleteCatalogInstance(ctx context.Context, instance *model.CatalogInstance) error {
	deploymentIdUrl, err := url.Parse(fmt.Sprintf("%s/%s", instance.Provider.Endpoint, instance.InstanceId))
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(&http.Request{
		Method: "DELETE",
		URL:    deploymentIdUrl,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete catalog instance: %s, %s", resp.Status, resp.Body)
	}
	return nil
}
