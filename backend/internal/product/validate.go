package product

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"stockflow/backend/internal/apperr"
)

var skuPattern = regexp.MustCompile("^[A-Z0-9-]+$")

func NormalizeSKU(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func ValidateCreate(in *CreateInput) []apperr.Detail {
	var details []apperr.Detail

	in.SKU = NormalizeSKU(in.SKU)
	in.Name = strings.TrimSpace(in.Name)

	switch {
	case in.SKU == "":
		details = append(details, apperr.Detail{
			Field:   "sku",
			Message: "is required",
		})
	case len(in.SKU) > 64:
		details = append(details, apperr.Detail{
			Field:   "sku",
			Message: "must be at most 64 characters",
		})
	case !skuPattern.MatchString(in.SKU):
		details = append(details, apperr.Detail{
			Field:   "sku",
			Message: "may contain only A-Z, 0-9 and -",
		})
	}

	switch {
	case in.Name == "":
		details = append(details, apperr.Detail{
			Field:   "name",
			Message: "is required",
		})
	case utf8.RuneCountInString(in.Name) > 200:
		details = append(details, apperr.Detail{
			Field:   "name",
			Message: "must be at most 200 characters",
		})
	}

	if in.Price == nil {
		details = append(details, apperr.Detail{
			Field:   "price",
			Message: "is required",
		})
	} else if *in.Price < 0 || *in.Price > MaxPrice {
		details = append(details, apperr.Detail{
			Field:   "price",
			Message: "must be between 0 and 1000000000",
		})
	}

	if in.Stock != nil && (*in.Stock < 0 || *in.Stock > MaxQty) {
		details = append(details, apperr.Detail{
			Field:   "stock",
			Message: "must be between 0 and 1000000",
		})
	}

	return details
}

func ValidateUpdate(in *UpdateInput) []apperr.Detail {
	var details []apperr.Detail

	if in.Name == nil {
		details = append(details, apperr.Detail{
			Field:   "name",
			Message: "is required",
		})
	} else {
		*in.Name = strings.TrimSpace(*in.Name)

		switch {
		case *in.Name == "":
			details = append(details, apperr.Detail{
				Field:   "name",
				Message: "is required",
			})
		case utf8.RuneCountInString(*in.Name) > 200:
			details = append(details, apperr.Detail{
				Field:   "name",
				Message: "must be at most 200 characters",
			})
		}
	}

	if in.Price == nil {
		details = append(details, apperr.Detail{
			Field:   "price",
			Message: "is required",
		})
	} else if *in.Price < 0 || *in.Price > MaxPrice {
		details = append(details, apperr.Detail{
			Field:   "price",
			Message: "must be between 0 and 1000000000",
		})
	}

	if in.IsActive == nil {
		details = append(details, apperr.Detail{
			Field:   "is_active",
			Message: "is required",
		})
	}

	return details
}
