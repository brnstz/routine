package main

import (
	"container/list"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/brnstz/routine/wikimg"
)

var (
	// Print an HTML div with the hex background
	fmtSpec = `<div style="background: %s; width=100%%">&nbsp;</div>`

	cache = newColorCache(50000)
)

// colorCache is a cache of recent URLs to imgResponse values. It expires older
// URLs once it contains the maximum number of values.
type colorCache struct {
	hmap  map[string]imgResponse
	count int
	max   int
	mutex sync.RWMutex
	exp   *list.List
}

// newColorCache creates colorCache that holds max items.
func newColorCache(max int) *colorCache {
	return &colorCache{
		hmap:  map[string]imgResponse{},
		count: 0,
		max:   max,
		mutex: sync.RWMutex{},
		exp:   list.New(),
	}
}

func (cc *colorCache) Add(url string, resp imgResponse) {
	// Lock the cache while we're adding
	cc.mutex.Lock()

	if cc.count >= cc.max {
		// If we've exceeded the max size, we must remove the oldest
		// element

		// Find the last element
		back := cc.exp.Back()

		// Remove it from the cache
		delete(cc.hmap, back.Value.(string))

		// Also remove it from the exp list
		cc.exp.Remove(back)
	} else {

		// Otherwise, we didn't remove anything so increment count
		cc.count++
	}

	// Add new url to be last to expire
	cc.exp.PushFront(url)

	// Save its value
	cc.hmap[url] = resp

	// Done locking
	cc.mutex.Unlock()
}

func (cc *colorCache) Get(url string) (imgResponse, bool) {
	cc.mutex.RLock()

	resp, ok := cc.hmap[url]

	cc.mutex.RUnlock()

	return resp, ok
}

// imgRequest is a request to get the first color from a URL
type imgRequest struct {
	p         *wikimg.Puller
	url       string
	responses chan imgResponse
}

// imgResponse contains the result of processing an imgRequest
type imgResponse struct {
	hex string
	err error
}

// worker takes imgRequests on the in channel, processes them and sends
// an imgResponse back on the request's channel
func worker(in chan *imgRequest) {
	for req := range in {
		var resp imgResponse

		// Check cache first
		resp, ok := cache.Get(req.url)

		if !ok {
			// It wasn't in the cache, so actually get it
			_, resp.hex, resp.err = req.p.FirstColor(req.url)

			cache.Add(req.url, resp)
		}

		// Send it back on our response channel
		req.responses <- resp
	}
}

func main() {
	var max, workers, buffer, port int

	flag.IntVar(&max, "max", 100, "maximum number of images per request")
	flag.IntVar(&workers, "workers", 25, "number of background workers")
	flag.IntVar(&buffer, "buffer", 10000, "size of buffered channels")
	flag.IntVar(&port, "port", 8000, "HTTP port to listen on")
	flag.Parse()

	// Create a buffered channel for communicating between image
	// puller loop and workers
	imgReqs := make(chan *imgRequest, buffer)

	// Create workers
	for i := 0; i < workers; i++ {
		go worker(imgReqs)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create a new image puller with our max
		p := wikimg.NewPuller(max)

		// Create a context with a 20 second timeout
		ctx, _ := context.WithTimeout(context.Background(), time.Second*20)

		// Set puller's Cancel channel, so it will be closed when the
		// context times out
		p.Cancel = ctx.Done()

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
