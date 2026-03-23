// Package errors provides custom error types for the application.
package errors

import "fmt"

// AppError represents an application-specific error with code, message, and HTTP status.
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
}

// Error implements the error interface for AppError.
func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Is implements the interface for errors.Is() comparison.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Predefined errors for common use cases.
var (
	// ErrNotFound represents a resource not found error (404).
	ErrNotFound = &AppError{
		Code:       "NOT_FOUND",
		Message:    "resource not found",
		HTTPStatus: 404,
	}

	// ErrBadRequest represents a bad request error (400).
	ErrBadRequest = &AppError{
		Code:       "BAD_REQUEST",
		Message:    "invalid request",
		HTTPStatus: 400,
	}

	// ErrInternal represents an internal server error (500).
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "internal server error",
		HTTPStatus: 500,
	}
)

// NewNotFoundError creates a new not found error with a custom message.
func NewNotFoundError(message string) *AppError {
	return &AppError{
		Code:       ErrNotFound.Code,
		Message:    message,
		HTTPStatus: ErrNotFound.HTTPStatus,
	}
}

// NewBadRequestError creates a new bad request error with a custom message.
func NewBadRequestError(message string) *AppError {
	return &AppError{
		Code:       ErrBadRequest.Code,
		Message:    message,
		HTTPStatus: ErrBadRequest.HTTPStatus,
	}
}

// NewInternalError creates a new internal error with a custom message.
func NewInternalError(message string) *AppError {
	return &AppError{
		Code:       ErrInternal.Code,
		Message:    message,
		HTTPStatus: ErrInternal.HTTPStatus,
	}
}
