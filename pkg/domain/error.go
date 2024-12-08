package domain

import "errors"

type TerminateError struct {
	Err error
}

func (e *TerminateError) Error() string {
	return e.Err.Error()
}

var TerminateErrorInstance = &TerminateError{Err: errors.New("terminate error")}
