package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/codegangsta/cli"
)

func playAction(context *cli.Context) {
	termStream, err := getTermStream(context.Args().First())
	if err != nil {
		logger.Fatal(err)
	}
	defer termStream.Close()
	fmt.Println("starting playback of recoding!\n")
	p := newPlayer(termStream, os.Stdout)
	if err := p.play(); err != nil {
		logger.Fatal(err)
	}
	fmt.Println("recoding playback complete!")
}

func getTermStream(path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("path must be specified")
	}
	if _, err := url.ParseRequestURI(path); err == nil {
		resp, err := http.Get(path)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
	logger.Debugf("opening file")
	return os.Open(path)
}

type playback struct {
	Pause   time.Duration
	Content []byte
}

// newPlayer returns a play to write the TTY stream to w which
// should also support a TTY termnal.
func newPlayer(termStream io.Reader, w *os.File) *player {
	return &player{
		s: termStream,
		w: w,
	}
}

type player struct {
	s io.Reader
	w *os.File
}

func (p *player) play() error {
	var (
		dec       = json.NewDecoder(p.s)
		playbacks = make(chan *playback, 64)
		wg        = &sync.WaitGroup{}
	)
	wg.Add(1)
	go p.writer(playbacks, wg)
	var prev *recoding
	for {
		var c *recoding
		if err := dec.Decode(&c); err != nil {
			if err == io.EOF {
				break
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
	close(playbacks)
	wg.Wait()
	return nil
}

func (p *player) writer(c chan *playback, wg *sync.WaitGroup) {
	defer wg.Done()
	for pc := range c {
		time.Sleep(pc.Pause)
		p.w.Write(pc.Content)
	}
}
