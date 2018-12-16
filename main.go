package main

import (
	"flag"
	"github.com/spacemeshos/poet-ref/service"
	"log"
)

var (
	rpcPort    uint
	wProxyPort uint
)

func init() {
	flag.UintVar(&rpcPort, "rpcport", 50052, "RPC server listening port")
	flag.UintVar(&wProxyPort, "wproxyport", 8080, "RPC server web proxy listening port")
	flag.Parse()
}

func main() {
	if err := service.Start(rpcPort, wProxyPort); err != nil {
		log.Fatal(err)
	}
}
