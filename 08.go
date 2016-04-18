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

	// cache is our global cache of urls to imgResponse values
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

// Add saves a url and its response to our cache
func (cc *colorCache) Add(url string, resp imgResponse) {
	// Lock the cache while we're adding
	cc.mutex.Lock()

	log.Println("adding", url)

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

// Get retrieves an imgResponse by its url, returning whether it was found or
// not as the second value
func (cc *colorCache) Get(url string) (imgResponse, bool) {
	cc.mutex.RLock()

	// Get it within read lock
	resp, ok := cc.hmap[url]

	cc.mutex.RUnlock()

	return resp, ok
}

// GetMulti feeds at most max values into the out channel, closing it when all
// possible entries have been exhausted (may be less than max)
func (cc *colorCache) GetMulti(max int, out chan imgResponse) {
	cc.mutex.RLock()

	i := 0
	for _, v := range cc.hmap {
		if i > max {
			break
		}

		// Skip results that were errors
		if v.err != nil {
			continue
		}

		i++
		out <- v
	}
	close(out)

	cc.mutex.RUnlock()
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

			// It wasn't in the cache, so actually get it and add it
			_, resp.hex, resp.err = req.p.FirstColor(req.url)
			cache.Add(req.url, resp)
		}

		// Send it back on our response channel
		req.responses <- resp
	}
}

func main() {
	var max, bgmax, workers, buffer, port int

	flag.IntVar(&max, "max", 300, "max number of images per HTTP request")
	flag.IntVar(&bgmax, "bgmax", 1000, "max images to pull on each background request")
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

	// Create background pull task
	go func() {

		// Loop forever
		for {

			// Create a new image puller with our bgmax
			p := wikimg.NewPuller(bgmax)

			// Since this is running in the background, we can have a much
			// longer timeout
			ctx, _ := context.WithTimeout(context.Background(), time.Minute*10)

			// Set puller's Cancel channel, so it will be closed when the
			// context times out
			p.Cancel = ctx.Done()

			// Create a channel for receiving responses in this background
			// process
			responses := make(chan imgResponse, max)

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
			}

			// Sleep for a bit until next iteration
			time.Sleep(30 * time.Minute)
		}

	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Create a channel
		responses := make(chan imgResponse, max)

		// Everybody gets a goroutine!
		go cache.GetMulti(max, responses)

		for resp := range responses {
			fmt.Fprintf(w, fmtSpec, resp.hex)
			fmt.Fprintln(w)
		}
	})

	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
