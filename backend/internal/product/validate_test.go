package product

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"stockflow/backend/internal/apperr"
)

func intPtr(v int) *int          { return &v }
func boolPtr(v bool) *bool       { return &v }
func stringPtr(v string) *string { return &v }

func TestValidateCreate(t *testing.T) {
	tests := []struct {
		name string
		in   CreateInput
		want []apperr.Detail
	}{
		{
			name: "valid input is normalized",
			in: CreateInput{
				SKU:   " brs-1 ",
				Name:  " Beras ",
				Price: intPtr(0),
			},
		},
		{
			name: "missing sku",
			in: CreateInput{
				SKU:   "",
				Name:  "Beras",
				Price: intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "sku", Message: "is required"},
			},
		},
		{
			name: "sku with invalid characters",
			in: CreateInput{
				SKU:   "AB C",
				Name:  "Beras",
				Price: intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "sku", Message: "may contain only A-Z, 0-9 and -"},
			},
		},
		{
			name: "sku too long",
			in: CreateInput{
				SKU:   strings.Repeat("A", 65),
				Name:  "Beras",
				Price: intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "sku", Message: "must be at most 64 characters"},
			},
		},
		{
			name: "blank name",
			in: CreateInput{
				SKU:   "BRS-1",
				Name:  "   ",
				Price: intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "name", Message: "is required"},
			},
		},
		{
			name: "name too long",
			in: CreateInput{
				SKU:   "BRS-1",
				Name:  strings.Repeat("N", 201),
				Price: intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "name", Message: "must be at most 200 characters"},
			},
		},
		{
			name: "price missing",
			in: CreateInput{
				SKU:  "BRS-1",
				Name: "Beras",
			},
			want: []apperr.Detail{
				{Field: "price", Message: "is required"},
			},
		},
		{
			name: "price below range",
			in: CreateInput{
				SKU:   "BRS-1",
				Name:  "Beras",
				Price: intPtr(-1),
			},
			want: []apperr.Detail{
				{Field: "price", Message: "must be between 0 and 1000000000"},
			},
		},
		{
			name: "price above range",
			in: CreateInput{
				SKU:   "BRS-1",
				Name:  "Beras",
				Price: intPtr(1_000_000_001),
			},
			want: []apperr.Detail{
				{Field: "price", Message: "must be between 0 and 1000000000"},
			},
		},
		{
			name: "negative stock",
			in: CreateInput{
				SKU:   "BRS-1",
				Name:  "Beras",
				Price: intPtr(0),
				Stock: intPtr(-1),
			},
			want: []apperr.Detail{
				{Field: "stock", Message: "must be between 0 and 1000000"},
			},
		},
		{
			name: "collects errors in field order",
			in: CreateInput{
				SKU:   "",
				Name:  "",
				Price: intPtr(-1),
			},
			want: []apperr.Detail{
				{Field: "sku", Message: "is required"},
				{Field: "name", Message: "is required"},
				{Field: "price", Message: "must be between 0 and 1000000000"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCreate(&tt.in)
			require.Equal(t, tt.want, got)

			if tt.name == "valid input is normalized" {
				require.Equal(t, "BRS-1", tt.in.SKU)
				require.Equal(t, "Beras", tt.in.Name)
			}
		})
	}
}

func TestValidateUpdate(t *testing.T) {
	tests := []struct {
		name string
		in   UpdateInput
		want []apperr.Detail
	}{
		{
			name: "valid input trims name",
			in: UpdateInput{
				Name:     stringPtr(" Beras "),
				Price:    intPtr(0),
				IsActive: boolPtr(false),
			},
		},
		{
			name: "empty input requires all fields",
			in:   UpdateInput{},
			want: []apperr.Detail{
				{Field: "name", Message: "is required"},
				{Field: "price", Message: "is required"},
				{Field: "is_active", Message: "is required"},
			},
		},
		{
			name: "blank name",
			in: UpdateInput{
				Name:     stringPtr("   "),
				Price:    intPtr(0),
				IsActive: boolPtr(true),
			},
			want: []apperr.Detail{
				{Field: "name", Message: "is required"},
			},
		},
		{
			name: "price out of range",
			in: UpdateInput{
				Name:     stringPtr("Beras"),
				Price:    intPtr(-1),
				IsActive: boolPtr(true),
			},
			want: []apperr.Detail{
				{Field: "price", Message: "must be between 0 and 1000000000"},
			},
		},
		{
			name: "name too long",
			in: UpdateInput{
				Name:     stringPtr(strings.Repeat("N", 201)),
				Price:    intPtr(0),
				IsActive: boolPtr(true),
			},
			want: []apperr.Detail{
				{Field: "name", Message: "must be at most 200 characters"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateUpdate(&tt.in)
			require.Equal(t, tt.want, got)

			if tt.name == "valid input trims name" {
				require.Equal(t, "Beras", *tt.in.Name)
			}
		})
	}
}

func TestValidateAdjust(t *testing.T) {
	tests := []struct {
		name string
		in   AdjustInput
		want []apperr.Detail
	}{
		{
			name: "valid input trims note",
			in: AdjustInput{
				Type: MovementIn,
				Qty:  intPtr(3),
				Note: stringPtr(" restock "),
			},
		},
		{
			name: "valid input converts blank note to nil",
			in: AdjustInput{
				Type: MovementOut,
				Qty:  intPtr(1),
				Note: stringPtr("   "),
			},
		},
		{
			name: "missing quantity",
			in: AdjustInput{
				Type: MovementIn,
			},
			want: []apperr.Detail{
				{Field: "qty", Message: "is required"},
			},
		},
		{
			name: "invalid type",
			in: AdjustInput{
				Type: "x",
				Qty:  intPtr(1),
			},
			want: []apperr.Detail{
				{Field: "type", Message: `must be "in" or "out"`},
			},
		},
		{
			name: "quantity below range",
			in: AdjustInput{
				Type: MovementIn,
				Qty:  intPtr(0),
			},
			want: []apperr.Detail{
				{Field: "qty", Message: "must be between 1 and 1000000"},
			},
		},
		{
			name: "quantity above range",
			in: AdjustInput{
				Type: MovementOut,
				Qty:  intPtr(1000001),
			},
			want: []apperr.Detail{
				{Field: "qty", Message: "must be between 1 and 1000000"},
			},
		},
		{
			name: "note too long",
			in: AdjustInput{
				Type: MovementIn,
				Qty:  intPtr(1),
				Note: stringPtr(strings.Repeat("N", 501)),
			},
			want: []apperr.Detail{
				{Field: "note", Message: "must be at most 500 characters"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateAdjust(&tt.in)
			require.Equal(t, tt.want, got)

			if tt.name == "valid input trims note" {
				require.Equal(t, "restock", *tt.in.Note)
			}
			if tt.name == "valid input converts blank note to nil" {
				require.Nil(t, tt.in.Note)
			}
		})
	}
}
