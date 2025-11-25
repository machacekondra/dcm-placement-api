package store

import (
	"gorm.io/gorm"
)

type Store interface {
	Close() error
	CatalogInstance() CatalogInstance
	CatalogItem() CatalogItem
	Resource() Resource
	ServiceProvider() ServiceProvider
}

type DataStore struct {
	db              *gorm.DB
	catalogInstance CatalogInstance
	catalogItems    CatalogItem
	resource        Resource
	serviceProvider ServiceProvider
}

func NewStore(db *gorm.DB) Store {
	return &DataStore{
		db:              db,
		catalogInstance: NewCatalogInstance(db),
		catalogItems:    NewCatalogItem(db),
		resource:        NewResource(db),
		serviceProvider: NewServiceProvider(db),
	}
}

func (s *DataStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (s *DataStore) CatalogInstance() CatalogInstance {
	return s.catalogInstance
}

func (s *DataStore) CatalogItem() CatalogItem {
	return s.catalogItems
}

func (s *DataStore) Resource() Resource {
	return s.resource
}

func (s *DataStore) ServiceProvider() ServiceProvider {
	return s.serviceProvider
}
