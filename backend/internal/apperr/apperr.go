package apperr

import "fmt"

type Detail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Error struct {
	Status    int      `json:"-"`
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Details   []Detail `json:"details,omitempty"`
	SKU       string   `json:"sku,omitempty"`
	Available *int     `json:"available,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func New(status int, code, msg string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: msg,
	}
}

func Validation(details []Detail) *Error {
	return &Error{
		Status:  400,
		Code:    "VALIDATION_FAILED",
		Message: "validation failed",
		Details: details,
	}
}

func InvalidJSON(msg string) *Error {
	return New(400, "INVALID_JSON", msg)
}

func NotFound(msg string) *Error {
	return New(404, "NOT_FOUND", msg)
}

func Conflict(code, msg string) *Error {
	return New(409, code, msg)
}

func InsufficientStock(sku string, available int) *Error {
	return &Error{
		Status:    409,
		Code:      "INSUFFICIENT_STOCK",
		Message:   "insufficient stock",
		SKU:       sku,
		Available: &available,
	}
}
