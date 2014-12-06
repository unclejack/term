package main

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

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
