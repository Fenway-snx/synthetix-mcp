package bootstrap

import (
	"gorm.io/gorm"
)

// Client provides read-only database queries for service startup data loading.
// All methods execute SELECT queries only. The caller is responsible for
// passing a read-only *gorm.DB handle (typically from ReaderWriter.Reader()).
type Client struct {
	db *gorm.DB
}

// NewClient creates a bootstrap Client backed by the given database handle.
func NewClient(db *gorm.DB) *Client {
	return &Client{db: db}
}
