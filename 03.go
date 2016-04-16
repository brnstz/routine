package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/brnstz/routine/wikimg"
)

var (
	// Print a blank line with the given 256 ANSI color
	fmtSpec = "\x1b[30;48;5;%dm%-80s\x1b[0m\n"
)

func main() {
	var max, workers, buffer int

	flag.IntVar(&max, "max", 100, "maximum number of images to retrieve")
	flag.IntVar(&workers, "workers", 50, "number of background workers")
	flag.IntVar(&buffer, "buffer", 10000, "buffer size of buffered channels")
	flag.Parse()

	// Create a new image puller with our max
	p := wikimg.NewPuller(max)

	// Create a buffered channel for communicating between image
	// puller loop and workers
	imgURLs := make(chan string, buffer)

	// Create another buffered channel to receive "done" messages from
	// workers
	done := make(chan bool, buffer)

	for i := 0; i < workers; i++ {
		go func() {
			for imgURL := range imgURLs {

				// Get the first color in this image
				color, _, err := wikimg.FirstColor(imgURL)
				if err != nil {
					log.Println(err)
					continue
				}

				// Print color to the terminal
				fmt.Printf(fmtSpec, color, "")
			}

			// Signal that we are done
			done <- true
		}()
	}

	// Loop to retrieve more images
	for {
		imgURL, err := p.Next()

		if err == wikimg.EndOfResults {
			// Break from loop when end of results is reached
			break

		} else if err != nil {
			// Log error and continue getting URLs
			log.Println(err)
			continue
		}

		// Send this imgURL to the channel
		imgURLs <- imgURL
	}

	// There are no more imgURLs to send, close the channel. This
	// will cause the range in the goroutines to complete, once any
	// buffered entries are exhausted.
	close(imgURLs)

	// Wait for done messages from each worker. We can't rely on
	// closing the channel, because none of the goroutines individually
	// knows when the entire process is complete. Instead, we count
	// and wait for a done message from each worker.
	for i := 0; i < workers; i++ {
		// Pull off the done channel but don't bother capturing the value
		<-done
	}

}
