//go:build darwin
// +build darwin

package tid

func Gettid() int {
	return -1
}
