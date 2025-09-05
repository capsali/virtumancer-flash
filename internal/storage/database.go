package storage

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Host represents a libvirt host connection configuration.
type Host struct {
	ID  string `gorm:"primaryKey" json:"id"`
	URI string `json:"uri"`
}

// InitDB initializes and returns a GORM database instance.
// This function MUST be exported (start with a capital letter) and be in the 'storage' package.
func InitDB(dataSourceName string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dataSourceName), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto-migrate the schema
	err = db.AutoMigrate(&Host{})
	if err != nil {
		return nil, err
	}

	return db, nil
}


