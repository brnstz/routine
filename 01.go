package main

import (
	"fmt"
	"log"
	"time"

	"github.com/brnstz/routine/wikimg"
	"github.com/mgutz/ansi"
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
			log.Println("retrieval error with %v: %v", imgURL, err)
			continue
		}

		go func() {
			counts, err := wikimg.TopColors(imgURL)
			if err != nil {
				fmt.Println("processing error with %v: %v", imgURL, err)
			}

			fmt.Println(
				ansi.Color("HELLO",
					fmt.Sprintf("black:%d", counts[0].XTermCode),
				),
			)

		}()
	}

	time.Sleep(1 * time.Hour)
}
