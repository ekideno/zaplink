package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type Error struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	if t.Code != "" && e.Code != t.Code {
		return false
	}
	return true
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Wrap(err error, status int, code, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{Status: status, Code: code, Message: message, Err: err}
}

func Wrapf(err error, status int, code, format string, args ...any) *Error {
	if err == nil {
		return nil
	}
	return &Error{Status: status, Code: code, Message: fmt.Sprintf(format, args...), Err: err}
}

func Status(err error) int {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Status != 0 {
		return appErr.Status
	}
	return http.StatusInternalServerError
}

func Message(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) && appErr.Message != "" {
		return appErr.Message
	}
	return http.StatusText(http.StatusInternalServerError)
}

func Code(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}

func From(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}
