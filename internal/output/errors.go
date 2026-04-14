package output

import (
	"errors"
	"fmt"
)

const SchemaVersion = "fakturownia-cli/v1alpha1"

type ErrorDetail struct {
	Class     string `json:"class"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
	Hint      string `json:"hint,omitempty"`
}

type WarningDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Pagination struct {
	Page     int  `json:"page"`
	PerPage  int  `json:"per_page"`
	Returned int  `json:"returned"`
	HasNext  bool `json:"has_next"`
}

type Meta struct {
	Command    string      `json:"command"`
	Profile    string      `json:"profile,omitempty"`
	DurationMS int64       `json:"duration_ms"`
	Pagination *Pagination `json:"pagination,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

type Envelope struct {
	SchemaVersion string          `json:"schema_version"`
	Status        string          `json:"status"`
	Data          any             `json:"data"`
	Errors        []ErrorDetail   `json:"errors"`
	Warnings      []WarningDetail `json:"warnings"`
	Meta          Meta            `json:"meta"`
}

type AppError struct {
	exitCode int
	detail   ErrorDetail
	rawBody  []byte
	cause    error
}

func (e *AppError) Error() string {
	return e.detail.Message
}

func (e *AppError) ExitCode() int {
	if e == nil || e.exitCode == 0 {
		return 9
	}
	return e.exitCode
}

func (e *AppError) Detail() ErrorDetail {
	if e == nil {
		return ErrorDetail{
			Class:   "internal",
			Code:    "internal_error",
			Message: "unknown error",
		}
	}
	return e.detail
}

func (e *AppError) RawBody() []byte {
	if e == nil {
		return nil
	}
	return e.rawBody
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *AppError) WithCause(err error) *AppError {
	e.cause = err
	return e
}

func (e *AppError) WithRawBody(body []byte) *AppError {
	e.rawBody = body
	return e
}

func NewAppError(exitCode int, class, code, message string, retryable bool, hint string) *AppError {
	return &AppError{
		exitCode: exitCode,
		detail: ErrorDetail{
			Class:     class,
			Code:      code,
			Message:   message,
			Retryable: retryable,
			Hint:      hint,
		},
	}
}

func Usage(code, message, hint string) *AppError {
	return NewAppError(2, "usage", code, message, false, hint)
}

func NotFound(code, message, hint string) *AppError {
	return NewAppError(3, "not_found", code, message, false, hint)
}

func AuthFailure(code, message, hint string) *AppError {
	return NewAppError(4, "auth", code, message, false, hint)
}

func Conflict(code, message, hint string) *AppError {
	return NewAppError(5, "conflict", code, message, false, hint)
}

func Network(code, message, hint string, retryable bool) *AppError {
	return NewAppError(6, "network", code, message, retryable, hint)
}

func Remote(code, message, hint string, retryable bool) *AppError {
	return NewAppError(8, "remote", code, message, retryable, hint)
}

func Internal(err error, message string) *AppError {
	appErr := NewAppError(9, "internal", "internal_error", message, false, "rerun with --json for a structured error envelope")
	if err != nil {
		appErr.cause = err
	}
	return appErr
}

func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err, err.Error())
}

type ExitError struct {
	Code int
}

func (e ExitError) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

func (e ExitError) ExitCode() int {
	if e.Code == 0 {
		return 9
	}
	return e.Code
}
