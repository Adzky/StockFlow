package order

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"stockflow/backend/internal/apperr"
	"stockflow/backend/internal/product"
)

func ValidateCreate(in *CreateInput) []apperr.Detail {
	var details []apperr.Detail

	in.Customer = strings.TrimSpace(in.Customer)

	if in.Customer == "" {
		details = append(details, apperr.Detail{
			Field:   "customer",
			Message: "is required",
		})
	} else if utf8.RuneCountInString(in.Customer) > 200 {
		details = append(details, apperr.Detail{
			Field:   "customer",
			Message: "must be at most 200 characters",
		})
	}

	switch {
	case len(in.Items) == 0:
		details = append(details, apperr.Detail{
			Field:   "items",
			Message: "must contain at least one item",
		})
	case len(in.Items) > MaxItems:
		details = append(details, apperr.Detail{
			Field:   "items",
			Message: "must contain at most 100 items",
		})
	}

	if len(in.Items) == 0 || len(in.Items) > MaxItems {
		return details
	}

	firstIndexBySKU := make(map[string]int)

	for i := range in.Items {
		item := &in.Items[i]

		item.SKU = product.NormalizeSKU(item.SKU)

		if item.SKU == "" {
			details = append(details, apperr.Detail{
				Field:   fmt.Sprintf("items[%d].sku", i),
				Message: "is required",
			})
		} else if firstIndex, exists := firstIndexBySKU[item.SKU]; exists {
			details = append(details, apperr.Detail{
				Field: fmt.Sprintf("items[%d].sku", i),
				Message: fmt.Sprintf(
					`duplicate sku %q (also in items[%d])`,
					item.SKU,
					firstIndex,
				),
			})
		} else {
			firstIndexBySKU[item.SKU] = i
		}

		if item.Qty == nil {
			details = append(details, apperr.Detail{
				Field:   fmt.Sprintf("items[%d].qty", i),
				Message: "is required",
			})
		} else if *item.Qty < 1 || *item.Qty > product.MaxQty {
			details = append(details, apperr.Detail{
				Field:   fmt.Sprintf("items[%d].qty", i),
				Message: "must be between 1 and 1000000",
			})
		}
	}

	return details
}
