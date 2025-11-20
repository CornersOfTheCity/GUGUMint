package db

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type MintRequest struct {
	ID        uint   `gorm:"primaryKey"`
	Hash      string `gorm:"uniqueIndex;size:66"`
	Address   string `gorm:"size:64"`
	Status    string `gorm:"size:32"`
	TxHash    string `gorm:"size:80"`
	CreatedAt int64
	UpdatedAt int64
}

func NewPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&MintRequest{}); err != nil {
		log.Printf("auto migrate error: %v", err)
	}

	return db, nil
}
