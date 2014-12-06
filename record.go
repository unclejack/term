package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/term"
	"github.com/kr/pty"
)

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
	if os.Getenv("RECORDING") == "true" {
		return fmt.Errorf("cannot start a recording inside a recording, too much inception...")
	}
	master, slave, err := pty.Open()
	if err != nil {
		return err
	}
	state, err := term.SetRawTerminal(os.Stdin.Fd())
	if err != nil {
		return err
	}
	defer func() {
		master.Close()
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
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}
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
