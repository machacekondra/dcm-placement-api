package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ServiceProvider struct {
	gorm.Model
	ID         uuid.UUID `gorm:"primaryKey;"`
	ResourceID uuid.UUID `gorm:"type:uuid;column:resource_id;not null"`
	Resource   Resource  `gorm:"foreignKey:ResourceID"`
	Name       string    `gorm:"column:name;not null"`
	Endpoint   string    `gorm:"column:endpoint;not null"`
}

type CatalogInstance struct {
	gorm.Model
	ID            uuid.UUID       `gorm:"primaryKey;"`
	Name          string          `gorm:"column:name;not null"`
	CatalogItemID uuid.UUID       `gorm:"column:catalog_item_id;not null"`
	CatalogItem   CatalogItem     `gorm:"foreignKey:CatalogItemID"`
	ProviderID    uuid.UUID       `gorm:"column:provider_id;not null"`
	Provider      ServiceProvider `gorm:"foreignKey:ProviderID"`
	Status        string          `gorm:"column:status;not null"`
	InstanceId    string          `gorm:"column:instance_id;not null"`
}

type CatalogItem struct {
	gorm.Model
	ID               uuid.UUID         `gorm:"primaryKey;"`
	Name             string            `gorm:"column:name;not null"`
	Parameters       JSONContent       `gorm:"column:parameters;type:jsonb;not null"`
	ResourceID       uuid.UUID         `gorm:"column:resource_id;not null"`
	Resource         Resource          `gorm:"foreignKey:ResourceID"`
	CatalogInstances []CatalogInstance `gorm:"foreignKey:CatalogItemID"`
}

type Resource struct {
	gorm.Model
	ID               uuid.UUID         `gorm:"primaryKey;"`
	ResourceType     string            `gorm:"column:resource_type;not null"`
	Content          JSONContent       `gorm:"column:content;type:jsonb;not null"`
	ServiceProviders []ServiceProvider `gorm:"foreignKey:ResourceID"`
	CatalogItems     []CatalogItem     `gorm:"foreignKey:ResourceID"`
}

type CatalogInstanceList []CatalogInstance
type CatalogItemList []CatalogItem
type ServiceProviderList []ServiceProvider
type ResourceList []Resource

type JSONContent map[string]interface{}

func (j JSONContent) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONContent) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, &j)
}
