package main

import (
	"flag"
	"fmt"

	"github.com/brnstz/routine/wikimg"
)

var (
	// Print a blank line with the given 256 ANSI color
	fmtSpec = "\x1b[30;48;5;%dm%-80s\x1b[0m\n"
)

// worker takes urls from the in channel, prints the color to the terminal and
// sends any errors back to the out channel.
func worker(p *wikimg.Puller, in chan string, out chan error) {
	for url := range in {

		// Get the first color in this image
		color, _, err := p.FirstColor(url)

		if err == nil {
			// Print color to the terminal when there's no
			// error
			fmt.Printf(fmtSpec, color, "")
		}

		// Send err (possibly nil) on the channel
		out <- err
	}
}

func main() {
	var max, workers, buffer int

	flag.IntVar(&max, "max", 100, "maximum number of images to retrieve")
	flag.IntVar(&workers, "workers", 25, "number of background workers")
	flag.IntVar(&buffer, "buffer", 10000, "size of buffered channels")
	flag.Parse()

	// Create a new image puller with our max
	p := wikimg.NewPuller(max)

	// Create a buffered channel for communicating between image
	// puller loop and workers
	imgURLs := make(chan string, buffer)

	// Create another buffered channel to receive errors from the worker.
	// A nil error represents successful processing.
	errs := make(chan error, buffer)

	for i := 0; i < workers; i++ {
		go worker(p, imgURLs, errs)
	}

	// Loop to retrieve more images
	for {
		imgURL, err := p.Next()

		if err == wikimg.EndOfResults {
			// Break from loop when end of results is reached
			break

		} else if err != nil {
			// Errors can occur before we send the request to the worker.
			// No problem, we can use the error channel here, too.
			errs <- err
			continue
		}

		// Send this imgURL to the channel
		imgURLs <- imgURL
	}

	// There are no more imgURLs to send, close the channel. This
	// will cause the range in the goroutines to complete, once any
	// buffered entries are exhausted.
	close(imgURLs)

	// Wait for all requests to complete and count errors
	errCount := 0
	for i := 0; i < max; i++ {
		err := <-errs
		if err != nil {
			errCount++
		}
	}

	fmt.Printf("Successfully processed %d/%d requests.\n", max-errCount, max)
}
