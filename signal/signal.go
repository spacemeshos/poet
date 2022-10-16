// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2015-2017 The Lightning Network Developers
// Copyright (c) 2017-2019 The Spacemesh developers

package signal

import (
	"log"
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
			log.Println("Received SIGINT (Ctrl+C)")
			s.shutdown()

		case <-s.requestShutdownChan:
			log.Println("Received shutdown request")
			s.shutdown()

		case <-s.quit:
			log.Println("Gracefully shutting down...")
			close(s.shutdownChan)
			return
		}
	}
}

// shutdown invokes the registered interrupt handlers, then signals the
// shutdownChan.
func (s *Signal) shutdown() {
	select {
	case <-s.quit:
		// already shutting down
		return
	case <-s.ShutdownRequestedChan:
	default:
		close(s.ShutdownRequestedChan)
	}

	if s.NumBlocking() > 0 {
		log.Println("Shutdown: tasks are blocking")
		return
	}

	// Signal the main interrupt handler to exit, and stop accept
	// post-facto requests.
	close(s.quit)
}

// RequestShutdown initiates a graceful shutdown from the application.
func (s *Signal) RequestShutdown() {
	s.requestShutdownChan <- struct{}{}
}

func (s *Signal) ShutdownRequested() bool {
	select {
	case <-s.ShutdownRequestedChan:
		return true
	default:
		return false
	}
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

	select {
	case <-s.ShutdownRequestedChan:
		if numBlocking == 0 {
			s.shutdown()
		}
	default:
	}
}

func (s *Signal) NumBlocking() int {
	s.blockingMtx.Lock()
	numBlocking := len(s.blocking)
	s.blockingMtx.Unlock()

	return numBlocking
}
