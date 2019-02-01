package playground

import (
	"flag"
)

var (
	n uint
)

func init() {
	flag.UintVar(&n, "n", 20, "Table size. T=2^n")
	flag.Parse()
}

func main() {

}
