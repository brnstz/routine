package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/brnstz/routine/wikimg"
)

var (
	// print a blank line with the given 256 ANSI color
	fmtSpec = "\x1b[30;48;5;%dm%-80s\x1b[0m\n"
)

func main() {
	var max int
	var count int
	flag.IntVar(&max, "max", 10, "maximum number of images to retrieve")
	flag.Parse()

	// Create a new image puller with our max
	p := wikimg.NewPuller(max)

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

		go func() {
			// Get the top color in this image
			color, err := wikimg.OneColor(imgURL)
			if err != nil {
				count++
				log.Println(err)
				return
			}

			// Print color to the terminal
			fmt.Printf(fmtSpec, color.XTermCode, "")
			count++
		}()

		for count < max {
			log.Printf("waiting %d < %d", count, max)
			time.Sleep(10 * time.Second)
		}

	}
}
