package main

import (
	"fmt"
	"time"

	"github.com/brnstz/routine/wikimg"
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
			//
			fmt.Println(err)
			continue
		}

		go func() {
			c, err := wikimg.TopColor(imgURL)
			if err != nil {
				fmt.Println(err)
			} else {
				fmt.Printf("#%02x%02x%02x\n", c.R, c.G, c.B)
				fmt.Printf("%v\n\n", imgURL)
			}
		}()
	}

	time.Sleep(1 * time.Hour)
}
