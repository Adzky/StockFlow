package order

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apperr"
)

func intPtr(v int) *int {
	return &v
}

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name string
		in   CreateInput
		want []apperr.Detail
	}{
		{
			name: "valid single item",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(2)},
				},
			},
		},
		{
			name: "empty items",
			in: CreateInput{
				Customer: "Budi",
				Items:    []ItemInput{},
			},
			want: []apperr.Detail{
				{Field: "items", Message: "must contain at least one item"},
			},
		},
		{
			name: "missing items",
			in: CreateInput{
				Customer: "Budi",
			},
			want: []apperr.Detail{
				{Field: "items", Message: "must contain at least one item"},
			},
		},
		{
			name: "too many items",
			in: CreateInput{
				Customer: "Budi",
				Items:    make101Items(),
			},
			want: []apperr.Detail{
				{Field: "items", Message: "must contain at most 100 items"},
			},
		},
		{
			name: "blank customer",
			in: CreateInput{
				Customer: " ",
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(1)},
				},
			},
			want: []apperr.Detail{
				{Field: "customer", Message: "is required"},
			},
		},
		{
			name: "customer too long",
			in: CreateInput{
				Customer: strings.Repeat("A", 201),
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(1)},
				},
			},
			want: []apperr.Detail{
				{Field: "customer", Message: "must be at most 200 characters"},
			},
		},
		{
			name: "blank item sku",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: " ", Qty: intPtr(1)},
				},
			},
			want: []apperr.Detail{
				{Field: "items[0].sku", Message: "is required"},
			},
		},
		{
			name: "missing quantity",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: "BRS-1"},
				},
			},
			want: []apperr.Detail{
				{Field: "items[0].qty", Message: "is required"},
			},
		},
		{
			name: "quantity below range",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(0)},
				},
			},
			want: []apperr.Detail{
				{Field: "items[0].qty", Message: "must be between 1 and 1000000"},
			},
		},
		{
			name: "quantity above range",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(1000001)},
				},
			},
			want: []apperr.Detail{
				{Field: "items[0].qty", Message: "must be between 1 and 1000000"},
			},
		},
		{
			name: "duplicate normalized sku",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: "ABC", Qty: intPtr(1)},
					{SKU: " abc ", Qty: intPtr(2)},
				},
			},
			want: []apperr.Detail{
				{
					Field:   "items[1].sku",
					Message: `duplicate sku "ABC" (also in items[0])`,
				},
			},
		},
		{
			name: "mixed case and padded sku",
			in: CreateInput{
				Customer: "Budi",
				Items: []ItemInput{
					{SKU: " brs-prm-5kg ", Qty: intPtr(1)},
				},
			},
		},
		{
			name: "customer and quantity errors are ordered",
			in: CreateInput{
				Customer: " ",
				Items: []ItemInput{
					{SKU: "BRS-1", Qty: intPtr(0)},
				},
			},
			want: []apperr.Detail{
				{Field: "customer", Message: "is required"},
				{Field: "items[0].qty", Message: "must be between 1 and 1000000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCreate(&tt.in)
			require.Equal(t, tt.want, got)
		})
	}

	var padded CreateInput
	padded.Customer = "Budi"
	padded.Items = []ItemInput{
		{SKU: " brs-prm-5kg ", Qty: intPtr(1)},
	}
	ValidateCreate(&padded)
	require.Equal(t, "BRS-PRM-5KG", padded.Items[0].SKU)
}

func make101Items() []ItemInput {
	items := make([]ItemInput, 101)
	for i := range items {
		items[i] = ItemInput{
			SKU: "ITEM-" + string(rune('A'+i%26)),
			Qty: intPtr(1),
		}
	}
	return items
}
