package product

import "time"

const (
	LowStockThreshold = 10
	MaxPrice          = 1_000_000_000
	MaxQty            = 1_000_000
	MovementIn        = "in"
	MovementOut       = "out"
)

type Product struct {
	ID             int        `json:"id"`
	SKU            string     `json:"sku"`
	Name           string     `json:"name"`
	Price          int        `json:"price"`
	Stock          int        `json:"stock"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
	LastMovementAt *time.Time `json:"last_movement_at"`
}

type Movement struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	Type      string    `json:"type"`
	Qty       int       `json:"qty"`
	Note      *string   `json:"note"`
	OrderID   *int      `json:"order_id"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateInput struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    *int   `json:"price"`
	Stock    *int   `json:"stock"`
	IsActive *bool  `json:"is_active"`
}

type UpdateInput struct {
	Name     *string `json:"name"`
	Price    *int    `json:"price"`
	IsActive *bool   `json:"is_active"`
}
