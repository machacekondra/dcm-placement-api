package model

import (
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Application struct {
	gorm.Model
	ID      uuid.UUID      `gorm:"primaryKey;"`
	Name    string         `gorm:"name;not null"`
	Service string         `gorm:"service;not null"`
	Zones   pq.StringArray `gorm:"type:text[]"`
	Tier    int            `gorm:"tier;not null"`
}

type ApplicationList []Application
