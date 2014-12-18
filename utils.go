package main

import (
	"os"
	"os/exec"
	"syscall"

	"github.com/docker/docker/pkg/term"
)

func setTermSize(master *os.File) {
	ws, err := term.GetWinsize(os.Stdin.Fd())
	if err != nil {
		logger.WithField("error", err).Error("getting STDIN window size")
	}
	if err := term.SetWinsize(master.Fd(), ws); err != nil {
		logger.WithField("error", err).Error("set master TTY window size")
	}
}

func forwardSignals(s chan os.Signal, cmd *exec.Cmd, master *os.File) {
	for sig := range s {
		switch sig {
		case syscall.SIGWINCH:
			setTermSize(master)
		default:
			proc := cmd.Process
			if proc != nil {
				proc.Signal(sig)
			}
		}
	}
}
