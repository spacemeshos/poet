package types

import (
	"errors"
	"fmt"
)

var (
	ErrChallengeInvalid = errors.New("challenge is invalid")
	ErrCouldNotVerify   = errors.New("could not verify the challenge")

	ErrChallengeAlreadySubmitted = errors.New("challenge is already submitted")
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
	err := ""
	for _, e := range e.errors {
		if err == "" {
			err = e.Error()
		} else {
			err = fmt.Sprintf("%v | %v", err, e)
		}
	}
	return err
}
