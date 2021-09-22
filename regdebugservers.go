// +build debug

package main

import (
	"github.com/spacemeshos/poet/rpc"
	"github.com/spacemeshos/poet/rpc/debug"
	"github.com/spacemeshos/smutil/log"
	"google.golang.org/grpc"
)

// startServer starts the RPC server.
func registerAdditionalRPCServers(g *grpc.Server) {
	log.Info("registering debug service")
	debugServer := rpc.NewDebugServer()
	debug.RegisterDebugServer(g, debugServer)
}
