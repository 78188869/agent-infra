package errors

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "not found error",
			err:      ErrNotFound,
			expected: "[NOT_FOUND] resource not found",
		},
		{
			name:     "bad request error",
			err:      ErrBadRequest,
			expected: "[BAD_REQUEST] invalid request",
		},
		{
			name:     "internal error",
			err:      ErrInternal,
			expected: "[INTERNAL_ERROR] internal server error",
		},
		{
			name: "custom error",
			err: &AppError{
				Code:       "CUSTOM",
				Message:    "something went wrong",
				HTTPStatus: 418,
			},
			expected: "[CUSTOM] something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    *AppError
		target error
		want   bool
	}{
		{
			name:   "same error type",
			err:    ErrNotFound,
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "different error type",
			err:    ErrNotFound,
			target: ErrBadRequest,
			want:   false,
		},
		{
			name:   "same code different instance",
			err:    NewNotFoundError("custom message"),
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "not an AppError",
			err:    ErrNotFound,
			target: errors.New("standard error"),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Is(tt.target)
			if got != tt.want {
				t.Errorf("Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrors_Is(t *testing.T) {
	// Test using standard errors.Is function
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("errors.Is(ErrNotFound, ErrNotFound) should return true")
	}
	if errors.Is(ErrNotFound, ErrBadRequest) {
		t.Error("errors.Is(ErrNotFound, ErrBadRequest) should return false")
	}

	customNotFound := NewNotFoundError("user not found")
	if !errors.Is(customNotFound, ErrNotFound) {
		t.Error("errors.Is(customNotFound, ErrNotFound) should return true")
	}
}

func TestNewNotFoundError(t *testing.T) {
	err := NewNotFoundError("user not found")

	if err.Code != "NOT_FOUND" {
		t.Errorf("Code = %v, want NOT_FOUND", err.Code)
	}
	if err.Message != "user not found" {
		t.Errorf("Message = %v, want 'user not found'", err.Message)
	}
	if err.HTTPStatus != 404 {
		t.Errorf("HTTPStatus = %v, want 404", err.HTTPStatus)
	}
}

func TestNewBadRequestError(t *testing.T) {
	err := NewBadRequestError("invalid email format")

	if err.Code != "BAD_REQUEST" {
		t.Errorf("Code = %v, want BAD_REQUEST", err.Code)
	}
	if err.Message != "invalid email format" {
		t.Errorf("Message = %v, want 'invalid email format'", err.Message)
	}
	if err.HTTPStatus != 400 {
		t.Errorf("HTTPStatus = %v, want 400", err.HTTPStatus)
	}
}

func TestNewInternalError(t *testing.T) {
	err := NewInternalError("database connection failed")

	if err.Code != "INTERNAL_ERROR" {
		t.Errorf("Code = %v, want INTERNAL_ERROR", err.Code)
	}
	if err.Message != "database connection failed" {
		t.Errorf("Message = %v, want 'database connection failed'", err.Message)
	}
	if err.HTTPStatus != 500 {
		t.Errorf("HTTPStatus = %v, want 500", err.HTTPStatus)
	}
}

func TestPredefinedErrors_HTTPStatus(t *testing.T) {
	tests := []struct {
		name           string
		err            *AppError
		expectedStatus int
	}{
		{"ErrNotFound", ErrNotFound, 404},
		{"ErrBadRequest", ErrBadRequest, 400},
		{"ErrInternal", ErrInternal, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.expectedStatus {
				t.Errorf("%s HTTPStatus = %v, want %v", tt.name, tt.err.HTTPStatus, tt.expectedStatus)
			}
		})
	}
}
