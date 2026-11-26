package httpx

import (
	"fmt"
	"net/http"
	"strconv"

	"stockflow/backend/internal/apperr"
)

type Paged[T any] struct {
	Data  []T `json:"data"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func QueryInt(
	r *http.Request,
	name string,
	def int,
	min int,
	max int,
) (int, error) {
	values, exists := r.URL.Query()[name]
	if !exists {
		return def, nil
	}

	value, err := strconv.Atoi("")
	if len(values) > 0 {
		value, err = strconv.Atoi(values[0])
	}
	if err != nil || value < min || value > max {
		return 0, apperr.Validation([]apperr.Detail{
			{
				Field:   name,
				Message: fmt.Sprintf("must be an integer between %d and %d", min, max),
			},
		})
	}

	return value, nil
}
