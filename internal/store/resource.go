package store

import (
	"context"

	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Resource interface {
	List(ctx context.Context) (model.ResourceList, error)
	Create(ctx context.Context, instance *model.Resource) (*model.Resource, error)
	Delete(ctx context.Context, resourceType string) error
	Update(ctx context.Context, instance model.Resource) (*model.Resource, error)
	Get(ctx context.Context, resourceType string) (*model.Resource, error)
}

type ResourceStore struct {
	db *gorm.DB
}

var _ Resource = (*ResourceStore)(nil)

func NewResource(db *gorm.DB) Resource {
	return &ResourceStore{db: db}
}

func (s *ResourceStore) List(ctx context.Context) (model.ResourceList, error) {
	var resourcelist model.ResourceList

	// Query with limit and offset
	// Also preloads all the related ServiceProviders for each Resource.
	result := s.db.Preload("ServiceProviders").Find(&resourcelist)
	if result.Error != nil {
		return nil, result.Error
	}
	return resourcelist, nil
}

func (s *ResourceStore) Delete(ctx context.Context, resourceType string) error {
	result := s.db.Where("resource_type = ?", resourceType).Delete(&model.Resource{})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *ResourceStore) Create(ctx context.Context, p *model.Resource) (*model.Resource, error) {
	result := s.db.Clauses(clause.Returning{}).Create(&p)
	if result.Error != nil {
		return nil, result.Error
	}

	return p, nil
}

func (s *ResourceStore) Get(ctx context.Context, resourceType string) (*model.Resource, error) {
	var p model.Resource
	result := s.db.Preload("ServiceProviders").Where("resource_type = ?", resourceType).First(&p)
	if result.Error != nil {
		return nil, result.Error
	}
	return &p, nil
}

func (s *ResourceStore) Update(ctx context.Context, instance model.Resource) (*model.Resource, error) {
	result := s.db.Save(&instance)
	if result.Error != nil {
		return nil, result.Error
	}
	return &instance, nil
}
