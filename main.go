package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/docker/docker/pkg/term"
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

func main() {
	app := cli.NewApp()
	app.Name = "rec"
	app.Version = "1"
	app.Author = "@crosbymichael"
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "play", Usage: "playback the recording in your term"},
	}
	app.Action = func(context *cli.Context) {
		path := context.Args().First()
		if path == "" {
			logger.Fatal("no path specified for recording")
		}
		if context.GlobalBool("play") {
			fmt.Println("starting playback of recoding!\n")
			if err := playbackTerm(path); err != nil {
				logger.Fatal(err)
			}
			fmt.Println("recoding complete!")
			return
		}
		if err := recordTerm(path); err != nil {
			logger.Fatal(err)
		}
	}
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
