package apperror

import "errors"

var (
	ErrNotFound          = errors.New("resource not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("resource already exists")
	ErrValidation        = errors.New("validation failed")
	ErrInsufficientFunds = errors.New("insufficient balance")
	ErrRateLimited       = errors.New("rate limited")
	ErrExternalService   = errors.New("external service error")
)

type AppError struct {
	Err     error
	Message string
	Fields  map[string]string
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(err error, message string) *AppError {
	return &AppError{Err: err, Message: message}
}

func Validation(fields map[string]string) *AppError {
	return &AppError{
		Err:     ErrValidation,
		Message: "Validation failed",
		Fields:  fields,
	}
}
