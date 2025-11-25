package store

import (
	"context"

	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CatalogItem interface {
	List(ctx context.Context) (model.CatalogItemList, error)
	Create(ctx context.Context, instance *model.CatalogItem) (*model.CatalogItem, error)
	Update(ctx context.Context, instance model.CatalogItem) (*model.CatalogItem, error)
	Delete(ctx context.Context, id string) error
	Get(ctx context.Context, id uuid.UUID) (*model.CatalogItem, error)
	GetByName(ctx context.Context, name string) (*model.CatalogItem, error)
}

type CatalogItemStore struct {
	db *gorm.DB
}

var _ CatalogItem = (*CatalogItemStore)(nil)

func NewCatalogItem(db *gorm.DB) CatalogItem {
	return &CatalogItemStore{db: db}
}

func (s *CatalogItemStore) GetByName(ctx context.Context, name string) (*model.CatalogItem, error) {
	var p model.CatalogItem
	result := s.db.Preload("Resource.ServiceProviders").Preload("Resource").Where("name = ?", name).First(&p)
	if result.Error != nil {
		return nil, result.Error
	}
	return &p, nil
}

func (s *CatalogItemStore) List(ctx context.Context) (model.CatalogItemList, error) {
	var templates model.CatalogItemList

	// Query with limit and offset
	tx := s.db.Model(&templates)
	result := tx.Preload("Resource").Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}
	return templates, nil
}

func (s *CatalogItemStore) Delete(ctx context.Context, name string) error {
	//FIXME by id
	var item model.CatalogItem
	result := s.db.Where("name = ?", name).First(&item)
	if result.Error != nil {
		return result.Error
	}
	deleteResult := s.db.Delete(&item)
	if deleteResult.Error != nil {
		return deleteResult.Error
	}
	return nil
}

func (s *CatalogItemStore) Create(ctx context.Context, p *model.CatalogItem) (*model.CatalogItem, error) {
	result := s.db.Clauses(clause.Returning{}).Create(&p)
	if result.Error != nil {
		return nil, result.Error
	}

	return p, nil
}

func (s *CatalogItemStore) Update(ctx context.Context, p model.CatalogItem) (*model.CatalogItem, error) {
	result := s.db.Save(&p)
	if result.Error != nil {
		return nil, result.Error
	}

	return &p, nil
}

func (s *CatalogItemStore) Get(ctx context.Context, id uuid.UUID) (*model.CatalogItem, error) {
	var p model.CatalogItem
	result := s.db.First(&p, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &p, nil
}
