//go:build !windows

package main

import "os/exec"

func hideChildProcessWindow(cmd *exec.Cmd) {
}
