//go:build windows

package main

import "os/exec"

func setupProcessGroup(cmd *exec.Cmd) {}
