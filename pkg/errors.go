package pkg

import (
	"errors"
	"strings"
)

type ErrorType int

const (
	// UnexpectedError
	// The error is not known
	UnexpectedError ErrorType = iota + 1
	// ValidationError
	// This error describe when the input image is not valid
	//
	ValidationError
	// UnknownResult
	//	This error appears when during validation strange error come up and we don't know the validation result.
	//	e.g.: communication errors
	//
	UnknownResult
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
	return UnexpectedError
}

func NewValidationFailedErr(err error) error {
	return NotaryError{
		code:    ValidationError,
		Message: "notary validation error",
		parent:  err,
	}
}

func NewUnknownResultErr(err error) error {
	return NotaryError{
		code:    UnknownResult,
		Message: "notary service unknown error",
		parent:  err,
	}
}
