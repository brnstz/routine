package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/brnstz/routine/wikimg"
)

var (
	// print a blank line with the given 256 ANSI color
	fmtSpec = "\x1b[30;48;5;%dm%-80s\x1b[0m\n"
)

func main() {
	var max, workers, buffer int

	flag.IntVar(&max, "max", 10, "maximum number of images to retrieve")
	flag.IntVar(&workers, "workers", 5, "number of background workers")
	flag.IntVar(&buffer, "buffer", 0, "buffer size of image channel")
	flag.Parse()

	// Create a new image puller with our max
	p := wikimg.NewPuller(max)

	// Create a buffered channel for communicating between image
	// puller loop and workers
	imgURLs := make(chan string, buffer)

	for i := 0; i < workers; i++ {
		go func() {
			for imgURL := range imgURLs {
				// Get the top color in this image
				color, err := wikimg.OneColor(imgURL)
				if err != nil {
					log.Println(err)
					continue
				}

				// Print color to the terminal
				fmt.Printf(fmtSpec, color.XTermCode, "")
			}
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

		imgURLs <- imgURL
	}

}
