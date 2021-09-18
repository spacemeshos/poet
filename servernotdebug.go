// +build !debug

package main

import (
	"google.golang.org/grpc"
)

// registerRPCServers does nothing in case debug tag was not supply.
func registerRPCServers(g *grpc.Server) {

}