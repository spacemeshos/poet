package service

import (
	"os"
	"os/signal"
)

var (
	// interruptChannel is used to receive SIGINT (Ctrl+C) signals.
	interruptChannel = make(chan os.Signal, 1)

	// shutdownRequestChannel is used to request the daemon to shutdown
	// gracefully, similar to when receiving SIGINT.
	shutdownRequestChannel = make(chan struct{})

	// quit is closed when instructing the main interrupt handler to exit.
	quit = make(chan struct{})

	// shutdownChannel is closed once the main interrupt handler exits.
	shutdownChannel = make(chan struct{})
)

func init() {
	signal.Notify(interruptChannel, os.Interrupt)
	go mainInterruptHandler()
}

func mainInterruptHandler() {
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
			println("Already shutting down...")
			return
		}
		isShutdown = true
		println("Shutting down...")

		// Signal the main interrupt handler to exit, and stop accept
		// post-facto requests.
		close(quit)
	}

	for {
		select {
		case <-interruptChannel:
			println("Received SIGINT (Ctrl+C).")
			shutdown()

		case <-shutdownRequestChannel:
			println("Received shutdown request.")
			shutdown()

		case <-quit:
			println("Gracefully shutting down...")
			close(shutdownChannel)
			return
		}
	}
}

// RequestShutdown initiates a graceful shutdown from the application.
func RequestShutdown() {
	shutdownRequestChannel <- struct{}{}
}

// ShutdownChannel returns the channel that will be closed once the main
// interrupt handler has exited.
func ShutdownChannel() <-chan struct{} {
	return shutdownChannel
}
