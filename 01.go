package main

import (
	"fmt"
	"image/color"
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
			c, err := wikimg.AvgColor(imgURL)
			if err != nil {
				fmt.Println(err)
			} else {
				rgba, ok := c.(color.RGBA)
				if !ok {
					panic(err)
				}

				fmt.Printf("#%02x%02x%02x\n", rgba.R, rgba.G, rgba.B)
				fmt.Printf("%v\n\n", imgURL)
			}
		}()
	}

	time.Sleep(1 * time.Hour)
}
