//go:build windows

package agent

import (
	"os"
	"os/exec"
)

// execBinary starts a new process and exits the current one.
// On Windows, syscall.Exec is not available, so we start a new process and exit.
func execBinary(binary string, args []string, env []string) error {
	cmd := exec.Command(binary, args[1:]...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// Exit current process
	os.Exit(0)
	return nil
}
