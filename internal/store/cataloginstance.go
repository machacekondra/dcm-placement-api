package store

import (
	"context"

	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogInstance interface {
	List(ctx context.Context) (model.CatalogInstanceList, error)
	Create(ctx context.Context, instance *model.CatalogInstance) (*model.CatalogInstance, error)
	Update(ctx context.Context, instance model.CatalogInstance) (*model.CatalogInstance, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*model.CatalogInstance, error)
}

type CatalogInstanceStore struct {
	db *gorm.DB
}

var _ CatalogInstance = (*CatalogInstanceStore)(nil)

func NewCatalogInstance(db *gorm.DB) CatalogInstance {
	return &CatalogInstanceStore{db: db}
}

func (s *CatalogInstanceStore) List(ctx context.Context) (model.CatalogInstanceList, error) {
	var instances model.CatalogInstanceList

	// Query with limit and offset
	tx := s.db.Model(&instances)
	result := tx.Preload("CatalogItem").Preload("CatalogItem.Resource").Preload("Provider").Find(&instances)
	if result.Error != nil {
		return nil, result.Error
	}
	return instances, nil
}

func (s *CatalogInstanceStore) Delete(ctx context.Context, id uuid.UUID) error {
	result := s.db.Delete(&model.CatalogInstance{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *CatalogInstanceStore) Create(ctx context.Context, instance *model.CatalogInstance) (*model.CatalogInstance, error) {
	result := s.db.Clauses(clause.Returning{}).Create(&instance)
	if result.Error != nil {
		return nil, result.Error
	}

	return instance, nil
}

func (s *CatalogInstanceStore) Update(ctx context.Context, instance model.CatalogInstance) (*model.CatalogInstance, error) {
	result := s.db.Save(&instance)
	if result.Error != nil {
		return nil, result.Error
	}

	return &instance, nil
}

func (s *CatalogInstanceStore) Get(ctx context.Context, id uuid.UUID) (*model.CatalogInstance, error) {
	var instance model.CatalogInstance
	tx := s.db.Model(&instance)
	result := tx.Preload("CatalogItem").Preload("CatalogItem.Resource").Preload("Provider").First(&instance, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &instance, nil
}
