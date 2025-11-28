//go:build !windows

package agent

import "syscall"

// execBinary replaces the current process with a new binary.
// On Unix, this uses syscall.Exec which atomically replaces the process.
func execBinary(binary string, args []string, env []string) error {
	return syscall.Exec(binary, args, env)
}
