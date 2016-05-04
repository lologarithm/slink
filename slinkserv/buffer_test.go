package slinkserv

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestBytePipe(t *testing.T) {
	pipe := NewBytePipe(20)
	go func() {
		pipe.Write(make([]byte, 5))
		pipe.Write(make([]byte, 6))
		pipe.Write(make([]byte, 7))
		pipe.Write(make([]byte, 8))
		log.Printf("Done writing all the bytes!")
	}()

	log.Printf("length: %d", pipe.Len())
	buf := make([]byte, 10)
	log.Printf("Trying first read")
	n := pipe.Read(buf)
	log.Printf("first read: %d", n)
	log.Printf("length: %d", pipe.Len())
	n2 := pipe.Read(buf)
	log.Printf("second read: %d", n2)
	if n+n2 == 15 {
		log.Printf("Two reads got all data.")
		return
	}
	n = pipe.Read(buf)
	log.Printf("Final read: %d", n)
}

func TestBytePipeLargeMessage(t *testing.T) {
	numBytes := 50
	var pipeCap uint32 = 10
	buffer := 20

	pipe := NewBytePipe(pipeCap)
	go func() {
		pipe.Write(make([]byte, numBytes))
		log.Printf("Wrote %d bytes to %d size pipe", numBytes, pipeCap)
	}()

	buf := make([]byte, buffer)
	total := 0
	for total < numBytes {
		total += pipe.Read(buf)
	}
	log.Printf("Read %d bytes using a %d buffer", total, buffer)
}

func BenchmarkChannelPipe(b *testing.B) {
	pipe := make(chan []byte, 100)
	totalBytes := b.N
	t := time.Now()
	total := 0
	n := (totalBytes * 128) / 1024

	b.ResetTimer()
	go func() {
		inbuf := make([]byte, 128)
		for i := 0; i < totalBytes; i++ {
			pipe <- inbuf
		}
	}()

	for i := 0; i < n; i++ {
		msg := <-pipe
		total += len(msg)
	}
	b.StopTimer()
	fmt.Printf("Rate: %.0f bytes/sec\n", float64(total)/time.Now().Sub(t).Seconds())
}

func BenchmarkBytePipe(b *testing.B) {
	pipe := NewBytePipe(12800)
	totalBytes := b.N
	t := time.Now()
	total := 0
	buf := make([]byte, 1024)
	n := (totalBytes * 128) / 1024

	b.ResetTimer()
	go func() {
		inbuf := make([]byte, 128)
		for i := 0; i < totalBytes; i++ {
			pipe.Write(inbuf)
		}
	}()

	for i := 0; i < n; i++ {
		total += pipe.Read(buf)
	}
	b.StopTimer()
	fmt.Printf("Rate: %.0f bytes/sec\n", float64(total)/time.Now().Sub(t).Seconds())
}
