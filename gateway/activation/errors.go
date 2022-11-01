package activation

import (
	"fmt"

	"github.com/spacemeshos/poet/shared"
)

var (
	ErrAtxNotFound = &AtxNotFoundError{}
	ErrTransport   = &TransportError{}
)

// AtxNotFoundError means that ATX with given ID was not found.
type AtxNotFoundError struct {
	id shared.ATXID
}

func (e *AtxNotFoundError) Error() string {
	return fmt.Sprintf("ATX %X not found", e.id)
}

func (e *AtxNotFoundError) Is(target error) bool {
	_, ok := target.(*AtxNotFoundError)
	return ok
}

// TransportError means there was a problem communicating
// with Activation Service.
type TransportError struct {
	// additional contextual information
	msg string
	// the source (if any) that caused the error
	source error
}

func (e *TransportError) Error() string {
	return fmt.Sprintf("transport error: %s (%v)", e.msg, e.source)
}

func (e *TransportError) Unwrap() error { return e.source }

func (e *TransportError) Is(target error) bool {
	_, ok := target.(*TransportError)
	return ok
}
