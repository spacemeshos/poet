//go:build windows
// +build windows

package tid

func Gettid() int {
	return -1
}
