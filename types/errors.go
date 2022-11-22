package types

import (
	"errors"
)

var (
	ErrChallengeInvalid = errors.New("challenge is invalid")
	ErrCouldNotVerify   = errors.New("could not verify the challenge")

	ErrChallengeAlreadySubmitted = errors.New("Challenge is already submitted")
)
