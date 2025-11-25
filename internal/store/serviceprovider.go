package store

import (
	"context"

	"github.com/dcm-project/dcm-placement-api/internal/store/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ServiceProvider interface {
	List(ctx context.Context) (model.ServiceProviderList, error)
	Create(ctx context.Context, instance *model.ServiceProvider) (*model.ServiceProvider, error)
	Update(ctx context.Context, instance model.ServiceProvider) (*model.ServiceProvider, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (*model.ServiceProvider, error)
}

type ServiceProviderStore struct {
	db *gorm.DB
}

var _ ServiceProvider = (*ServiceProviderStore)(nil)

func NewServiceProvider(db *gorm.DB) ServiceProvider {
	return &ServiceProviderStore{db: db}
}

func (s *ServiceProviderStore) List(ctx context.Context) (model.ServiceProviderList, error) {
	var providerservice model.ServiceProviderList
	// Query with limit and offset
	tx := s.db.Model(&providerservice)
	result := tx.Find(&providerservice)
	if result.Error != nil {
		return nil, result.Error
	}
	return providerservice, nil
}

func (s *ServiceProviderStore) Delete(ctx context.Context, id uuid.UUID) error {
	result := s.db.Delete(&model.ServiceProvider{}, id)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *ServiceProviderStore) Create(ctx context.Context, p *model.ServiceProvider) (*model.ServiceProvider, error) {
	result := s.db.Clauses(clause.Returning{}).Create(&p)
	if result.Error != nil {
		return nil, result.Error
	}
	return p, nil
}

func (s *ServiceProviderStore) Update(ctx context.Context, p model.ServiceProvider) (*model.ServiceProvider, error) {
	result := s.db.Save(&p)
	if result.Error != nil {
		return nil, result.Error
	}
	return &p, nil
}

func (s *ServiceProviderStore) Get(ctx context.Context, id uuid.UUID) (*model.ServiceProvider, error) {
	var p model.ServiceProvider
	result := s.db.First(&p, id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &p, nil
}
