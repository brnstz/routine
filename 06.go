package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/brnstz/routine/wikimg"
)

var (
	// Print an HTML div with the hex background
	fmtSpec = `<div style="background: %s; width=100%%">&nbsp;</div>`
)

type imgRequest struct {
	p         *wikimg.Puller
	url       string
	responses chan imgResponse
	cancel    chan struct{}
}

type imgResponse struct {
	hex string
	err error
}

func main() {
	var max, workers, buffer, port int

	flag.IntVar(&max, "max", 100, "maximum number of images per request")
	flag.IntVar(&workers, "workers", 50, "number of background workers")
	flag.IntVar(&buffer, "buffer", 10000, "size of buffered channels")
	flag.IntVar(&port, "port", 8000, "HTTP port to listen on")
	flag.Parse()

	// Create a buffered channel for communicating between image
	// puller loop and workers
	imgReqs := make(chan *imgRequest, buffer)

	for i := 0; i < workers; i++ {
		go func() {
			for req := range imgReqs {
				// Create a local channel for receiving the response
				localResponse := make(chan imgResponse, 1)
				go func() {
					// Get the first color in this image
					_, hex, err := req.p.FirstColor(req.url)

					localResponse <- imgResponse{
						hex: hex,
						err: err,
					}
				}()

				// Either get the response or timeout
				select {
				case resp := <-localResponse:
					// We got it in time, send it back to client
					req.responses <- resp
				case <-req.cancel:
					break

				case <-time.After(2 * time.Second):
					// We timed out. Send an error back to the client
					// and cancel our request.
					close(req.cancel)
					req.responses <- imgResponse{
						err: errors.New("timeout"),
					}
				}
			}
		}()
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create a new image puller with our max
		p := wikimg.NewPuller(max)

		cancel := make(chan struct{})

		p.Cancel = cancel

		// Create a channel for receiving responses specific
		// to this HTTP request
		responses := make(chan imgResponse, max)

		// Assert our writer to a flusher, so we can stream line by line
		f, ok := w.(http.Flusher)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Loop to retrieve more images
		for {
			imgURL, err := p.Next()

			if err == wikimg.EndOfResults {
				// Break from loop when end of results is reached
				break

			} else if err != nil {
				// Send error on the response channel and continue
				responses <- imgResponse{err: err}
				continue
			}

			// Create request and send on the global channel
			imgReqs <- &imgRequest{
				p:         p,
				url:       imgURL,
				responses: responses,
				cancel:    cancel,
			}
		}

		for i := 0; i < max; i++ {
			// Read a response from the channel
			resp := <-responses

			// If there's an error, just log it on the server
			if resp.err != nil {
				log.Println(resp.err)
				continue
			}

			// Write a line of color
			fmt.Fprintf(w, fmtSpec, resp.hex)
			fmt.Fprintln(w)
			f.Flush()
		}
	})

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
