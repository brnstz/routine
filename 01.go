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
			counts, err := wikimg.TopColors(imgURL)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(imgURL)
			for k, v := range counts {
				fmt.Println(k, v.Hex)
			}
			fmt.Println()
		}()
	}

	time.Sleep(1 * time.Hour)
}
