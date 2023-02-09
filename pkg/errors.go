package pkg

import (
	"errors"
	"strings"
)

type ErrorType int

const (
	UnknownError ErrorType = iota + 1
	ValidationError
	ServiceUnavailableError
)

type NotaryError struct {
	code    ErrorType
	Message string
	parent  error
}

func (e NotaryError) Error() string {
	builder := strings.Builder{}
	builder.WriteString(e.Message)

	if e.parent != nil {
		builder.WriteString(": ")
		builder.WriteString(e.parent.Error())
	}
	return builder.String()
}

func (e NotaryError) Is(err error) bool {
	if customErr, ok := err.(NotaryError); ok {
		return e.code == customErr.code
	}
	return false
}

func ErrorCode(e error) ErrorType {
	var customErr NotaryError
	if found := errors.As(e, &customErr); found {
		return customErr.code
	}
	return UnknownError
}

func NewValidationError(err error) error {
	return NotaryError{
		code:    ValidationError,
		Message: "notary validation error",
		parent:  err,
	}
}

func NewServiceUnavailableError(err error) error {
	return NotaryError{
		code:    ServiceUnavailableError,
		Message: "notary service unavailable error",
		parent:  err,
	}
}
