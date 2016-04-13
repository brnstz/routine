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

type imgReq struct {
	imgURL string
	resp   chan imgResp
}

// imgResp
type imgResp struct {
	cc  wikimg.ColorCount
	err error
}

func main() {
	var max, workers, buffer int

	flag.IntVar(&max, "max", 100, "maximum number of images to retrieve")
	flag.IntVar(&workers, "workers", 100, "number of background workers")
	flag.IntVar(&buffer, "buffer", 100, "buffer size of image channel")
	flag.Parse()

	// Create a new image puller with our max
	p := wikimg.NewPuller(max)

	// Create buffered channels for communicating between image
	// puller loop and workers
	imgReqs := make(chan imgReq, buffer)

	for i := 0; i < workers; i++ {
		go func() {
			for req := range imgReqs {
				log.Println("incoming", req)
				// Get the top color in this image
				cc, err := wikimg.OneColor(req.imgURL)
				log.Println("got color", cc, err)

				// Send response over the channel
				req.resp <- imgResp{cc: cc, err: err}
				log.Println("sent color", cc, err)
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

		req := imgReq{
			imgURL: imgURL,
		}

		log.Println("outgoing", req)
		// Send request
		imgReqs <- req
		log.Println("sent outgoing", req)

		// Wait for response
		resp := <-req.resp
		log.Println("got response", resp)

		if resp.err != nil {
			log.Println(resp.err)
			continue
		}

		// Print color to the terminal
		fmt.Printf(fmtSpec, resp.cc.XTermCode, "")
	}
}
