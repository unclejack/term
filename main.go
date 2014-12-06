package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

type recoding struct {
	Timestamp time.Time `json:"timestamp"`
	Content   []byte    `json:"content"`
}

func newRecorder(f io.WriteCloser) io.WriteCloser {
	return &recorder{
		output:  f,
		encoder: json.NewEncoder(f),
	}
}

type recorder struct {
	output  io.WriteCloser
	encoder *json.Encoder
}

func (r *recorder) Write(p []byte) (int, error) {
	c := recoding{
		Timestamp: time.Now(),
		Content:   p,
	}
	if err := r.encoder.Encode(c); err != nil {
		return -1, err
	}
	return len(p), nil
}

func (r *recorder) Close() error {
	return r.output.Close()
}

func recordTerm(path string) error {
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	r := newRecorder(f)
	defer r.Close()
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
	go io.Copy(io.MultiWriter(os.Stdout, r), master)
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}

func playbackTerm(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := json.NewDecoder(f)

	playbacks := make(chan *playback, 64)
	wg := &sync.WaitGroup{}
	defer func() {
		close(playbacks)
		wg.Wait()
	}()
	go playbackWriter(playbacks, wg)
	var prev *recoding
	for {
		var c *recoding
		if err := dec.Decode(&c); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		var d time.Duration
		if prev != nil {
			d = c.Timestamp.Sub(prev.Timestamp)
		}
		playbacks <- &playback{
			Pause:   d,
			Content: c.Content,
		}
		prev = c
	}
}

type playback struct {
	Pause   time.Duration
	Content []byte
}

func playbackWriter(c chan *playback, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	for p := range c {
		time.Sleep(p.Pause)
		os.Stdout.Write(p.Content)
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
