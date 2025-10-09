package main

import (
	"fmt"
	"time"
)

func main() {
	start := time.Now()
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v", time.Since(start))
}

func sig(after time.Duration) <-chan interface{} {
	c := make(chan interface{}, 1)
	go func() {
		defer close(c)
		time.Sleep(after)
	}()
	return c
}

func or(channels ...<-chan interface{}) <-chan interface{} {
	orDone := make(chan interface{})
	switch len(channels) {
	case 0:
		return orDone
	case 1:
		return channels[0]
	case 2:
		go func() {
			defer close(orDone)
			select {
			case <-channels[0]:
			case <-channels[1]:
			}
		}()
		return orDone
	default:
		go func() {
			defer close(orDone)
			select {
			case <-channels[0]:
			case <-or(channels[1:]...):
			}
		}()
		return orDone
	}

}
