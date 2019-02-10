// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2015-2016 The Decred developers
// Copyright (c) 2015-2017 The Lightning Network Developers
// Copyright (c) 2017-2019 The Spacemesh developers

package signal

import (
	"os"
	"os/signal"
)

type Signal struct  {
	// interruptChannel is used to receive SIGINT (Ctrl+C) signals.
	interruptChannel chan os.Signal

	// shutdownRequestChannel is used to request the daemon to shutdown
	// gracefully, similar to when receiving SIGINT.
	shutdownRequestChannel chan struct{}

	// quit is closed when instructing the main interrupt handler to exit.
	quit chan struct{}

	// shutdownChannel is closed once the main interrupt handler exits.
	shutdownChannel chan struct{}
}

func NewSignal() *Signal {
	s := new(Signal)
	s.interruptChannel = make(chan os.Signal, 1)
	s.shutdownRequestChannel = make(chan struct{})
	s.quit = make(chan struct{})
	s.shutdownChannel = make(chan struct{})

	signal.Notify(s.interruptChannel, os.Interrupt)
	go s.mainInterruptHandler()

	return s
}

func (s *Signal) mainInterruptHandler() {
	// isShutdown is a flag which is used to indicate whether or not
	// the shutdown signal has already been received and hence any future
	// attempts to add a new interrupt handler should invoke them
	// immediately.
	var isShutdown bool

	// shutdown invokes the registered interrupt handlers, then signals the
	// shutdownChannel.
	shutdown := func() {
		// Ignore more than one shutdown signal.
		if isShutdown {
			log.Infof("Already shutting down...")
			return
		}
		isShutdown = true
		log.Infof("Shutting down...")

		// Signal the main interrupt handler to exit, and stop accept
		// post-facto requests.
		close(s.quit)
	}

	for {
		select {
		case <-s.interruptChannel:
			log.Infof("Received SIGINT (Ctrl+C).")
			shutdown()

		case <-s.shutdownRequestChannel:
			log.Infof("Received shutdown request.")
			shutdown()

		case <-s.quit:
			log.Infof("Gracefully shutting down...")
			close(s.shutdownChannel)
			return
		}
	}
}

// RequestShutdown initiates a graceful shutdown from the application.
func (s *Signal) RequestShutdown() {
	s.shutdownRequestChannel <- struct{}{}
}

// ShutdownChannel returns the channel that will be closed once the main
// interrupt handler has exited.
func (s *Signal) ShutdownChannel() <-chan struct{} {
	return s.shutdownChannel
}
