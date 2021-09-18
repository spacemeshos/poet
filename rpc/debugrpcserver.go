// +build debug

package rpc

import (
	"github.com/spacemeshos/poet/rpc/debug"
	"github.com/spacemeshos/poet/signal"
	"github.com/spacemeshos/smutil/log"
	"golang.org/x/net/context"
)

// debugServer is a gRPC, RPC front end for the poet used for debugging and not meant to be turned added
// on production, it is activated using build tag `debug`
type debugServer struct{
	debug.DebugServer
}

// NewDebugServer creates and returns a new instance of the rpcServer.
func NewDebugServer() *debugServer {
	return &debugServer{}
}

func (d *debugServer) Shutdown(context.Context, *debug.ShutdownRequest) (*debug.ShutdownResponse, error) {
	log.Panic("shutting down ungracefully!")
	// TODO: this doesn't work #@!
	sig := signal.NewSignal()
	sig.RequestShutdown()
	return &debug.ShutdownResponse{}, nil
}
