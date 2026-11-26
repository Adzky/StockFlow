package order

import "time"

const (
	MaxItems             = 100
	StatusCreated        = "CREATED"
	StatusCancelled      = "CANCELLED"
	CodeAlreadyCancelled = "ORDER_ALREADY_CANCELLED"
)

type ItemInput struct {
	SKU string `json:"sku"`
	Qty *int   `json:"qty"`
}

type CreateInput struct {
	Customer string      `json:"customer"`
	Items    []ItemInput `json:"items"`
}

type Item struct {
	ID        int    `json:"id"`
	SKU       string `json:"sku"`
	Name      string `json:"name"`
	Qty       int    `json:"qty"`
	UnitPrice int    `json:"unit_price"`
	LineTotal int64  `json:"line_total"`
}

type Order struct {
	ID          int        `json:"id"`
	Customer    string     `json:"customer"`
	Status      string     `json:"status"`
	Total       int64      `json:"total"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	Items       []Item     `json:"items"`
}

type Summary struct {
	ID          int        `json:"id"`
	Customer    string     `json:"customer"`
	Status      string     `json:"status"`
	Total       int64      `json:"total"`
	CreatedAt   time.Time  `json:"created_at"`
	CancelledAt *time.Time `json:"cancelled_at"`
	ItemCount   int        `json:"item_count"`
}
