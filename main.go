package main

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

var logger = logrus.New()

func main() {
	app := cli.NewApp()
	app.Name = "term"
	app.Version = "1"
	app.Author = "@crosbymichael"
	app.Commands = []cli.Command{
		{
			Name:   "rec",
			Usage:  "record your terminal to the specified file",
			Action: recordAction,
		},
		{
			Name:   "play",
			Usage:  "play the specified recording from a location",
			Action: playAction,
		},
	}
	if err := app.Run(os.Args); err != nil {
		logger.Fatal(err)
	}
}
