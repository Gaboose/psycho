package main

import (
	"fmt"
	"io"
	"time"
)

type ReadRateLimiter struct {
	reader         io.Reader
	bytesPerSecond int64
	blockuntil     time.Time
}

func NewReadRateLimiter(reader io.Reader, bytesPerSecond int64) *ReadRateLimiter {
	return &ReadRateLimiter{
		reader:         reader,
		bytesPerSecond: bytesPerSecond,
	}
}

func (rrl *ReadRateLimiter) Read(p []byte) (int, error) {
	now := time.Now()
	if rrl.blockuntil.After(now) {
		time.Sleep(now.Sub(rrl.blockuntil))
	}

	n, err := rrl.reader.Read(p)

	rrl.blockuntil = now.Add(time.Duration(int64(n) * int64(time.Second) / rrl.bytesPerSecond))
	fmt.Println(rrl.blockuntil)
	return n, err
}
