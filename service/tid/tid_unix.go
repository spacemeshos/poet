//go:build linux
// +build linux

package tid

import "golang.org/x/sys/unix"

func Gettid() int {
	return unix.Gettid()
}
