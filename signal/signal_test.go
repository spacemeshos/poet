package signal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignal(t *testing.T) {
	req := require.New(t)
	s := NewSignal()

	go func() {
		time.Sleep(100 * time.Millisecond)
		s.RequestShutdown()
	}()

	select {
	case <-s.ShutdownChannel():
	case <-time.After(200 * time.Millisecond):
		req.Fail("timeout")
	}
}

func TestSignal_Blocking(t *testing.T) {
	req := require.New(t)
	s := NewSignal()

	go func() {
		time.Sleep(50 * time.Millisecond)
		unblock := s.BlockShutdown()

		time.Sleep(50 * time.Millisecond)
		s.RequestShutdown()

		time.Sleep(50 * time.Millisecond)
		unblock()
	}()

	select {
	case <-s.ShutdownChannel():
	case <-time.After(500 * time.Millisecond):
		req.Fail("timeout")
	}
}
