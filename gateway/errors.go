package gateway

import (
	"strings"
)

type MultiError struct {
	errors []error
}

func NewMultiError(errors []error) *MultiError {
	return &MultiError{
		errors: errors,
	}
}

func (e *MultiError) Error() string {
	// concatenate errors
	var b strings.Builder
	if len(e.errors) > 0 {
		b.WriteString(e.errors[0].Error())
		for _, e := range e.errors[1:] {
			b.WriteString(" | ")
			b.WriteString(e.Error())
		}
	}
	return b.String()
}
