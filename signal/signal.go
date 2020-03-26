// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2015-2017 The Lightning Network Developers
// Copyright (c) 2017-2019 The Spacemesh developers

package signal

import (
	"github.com/spacemeshos/smutil/log"
	"os"
	"os/signal"
	"sync"
)

type Signal struct {
	// interruptChan is used to receive SIGINT (Ctrl+C) signals.
	interruptChan chan os.Signal

	// requestShutdownChan is used to request the daemon to shutdown
	// gracefully, similar to when receiving SIGINT.
	requestShutdownChan chan struct{}

	// quit is closed when instructing the main interrupt handler to exit.
	quit chan struct{}

	// shutdownChan is closed once the main interrupt handler exits.
	shutdownChan chan struct{}

	// ShutdownStarted is a flag which is used to indicate whether or not
	// the shutdown signal has already been received and hence any future
	// attempts to add a new interrupt handler should invoke them
	// immediately.
	ShutdownStarted bool

	ShutdownRequested     bool
	ShutdownRequestedChan chan struct{}

	blocking       map[int]bool
	blockingNextID int
	blockingMtx    sync.Mutex
}

func NewSignal() *Signal {
	s := new(Signal)
	s.interruptChan = make(chan os.Signal, 1)
	s.quit = make(chan struct{})
	s.shutdownChan = make(chan struct{})
	s.requestShutdownChan = make(chan struct{})
	s.ShutdownRequestedChan = make(chan struct{})
	s.blocking = make(map[int]bool)

	signal.Notify(s.interruptChan, os.Interrupt)
	go s.mainInterruptHandler()

	return s
}

func (s *Signal) mainInterruptHandler() {
	for {
		select {
		case <-s.interruptChan:
			log.Info("Received SIGINT (Ctrl+C)")
			s.shutdown()

		case <-s.requestShutdownChan:
			log.Info("Received shutdown request")
			s.shutdown()

		case <-s.quit:
			log.Info("Gracefully shutting down...")
			close(s.shutdownChan)
			return
		}
	}
}

// shutdown invokes the registered interrupt handlers, then signals the
// shutdownChan.
func (s *Signal) shutdown() {
	if !s.ShutdownRequested {
		s.ShutdownRequested = true
		close(s.ShutdownRequestedChan)
	}

	// Ignore more than one shutdown signal.
	if s.ShutdownStarted {
		log.Info("Already shutting down...")
		return
	}

	if s.NumBlocking() > 0 {
		log.Info("Shutdown: tasks are blocking")
		return
	}

	s.ShutdownStarted = true

	// Signal the main interrupt handler to exit, and stop accept
	// post-facto requests.
	close(s.quit)
}

// RequestShutdown initiates a graceful shutdown from the application.
func (s *Signal) RequestShutdown() {
	s.requestShutdownChan <- struct{}{}
}

// ShutdownChannel returns the channel that will be closed once the main
// interrupt handler has exited.
func (s *Signal) ShutdownChannel() <-chan struct{} {
	return s.shutdownChan
}

func (s *Signal) BlockShutdown() func() {
	s.blockingMtx.Lock()

	s.blocking[s.blockingNextID] = true
	next := s.blockingNextID
	s.blockingNextID++

	s.blockingMtx.Unlock()

	return func() {
		s.unblockShutdown(next)
	}
}

func (s *Signal) unblockShutdown(id int) {
	s.blockingMtx.Lock()
	delete(s.blocking, id)
	numBlocking := len(s.blocking)
	s.blockingMtx.Unlock()

	if numBlocking == 0 && s.ShutdownRequested {
		s.shutdown()
	}
}

func (s *Signal) NumBlocking() int {
	s.blockingMtx.Lock()
	numBlocking := len(s.blocking)
	s.blockingMtx.Unlock()

	return numBlocking
}
