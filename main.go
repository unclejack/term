package main

import (
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/term"
	"github.com/kr/pty"
)

var logger = logrus.New()

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

func recordTerm() error {
	master, slave, err := pty.Open()
	if err != nil {
		return err
	}
	state, err := term.SetRawTerminal(os.Stdin.Fd())
	if err != nil {
		return err
	}
	defer func() {
		//slave.Close()
		//master.Close()
		term.RestoreTerminal(os.Stdin.Fd(), state)
	}()
	s := make(chan os.Signal, 32)
	signal.Notify(s)

	cmd := exec.Command(os.Getenv("SHELL"))
	cmd.Env = append(os.Environ(), "RECORDING=true")
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	setTermSize(master)
	go forwardSignals(s, cmd, master)
	go io.Copy(master, os.Stdin)
	go io.Copy(os.Stdout, master)
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	close(s)
	return nil
}

func main() {
	app := cli.NewApp()
	app.Name = "rec"
	app.Version = "1"
	app.Author = "@crosbymichael"
	app.Action = func(context *cli.Context) {
		if err := recordTerm(); err != nil {
			logger.Fatal(err)
		}
	}
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
