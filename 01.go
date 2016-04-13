package main

import (
	"fmt"
	"log"

	"github.com/brnstz/routine/wikimg"
)

var (
	// print a blank line with the given 256 ANSI color
	fmtSpec = "\x1b[30;48;5;%dm%-80s\x1b[0m\n"
)

func main() {
	// Let's start simple and pull 10 images
	p := wikimg.NewPuller(10)

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
				log.Println(err)
				return
			}

			// Print color to the terminal
			fmt.Printf(fmtSpec, color.XTermCode, "")
		}()
	}
}
