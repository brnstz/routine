// Package wikimg can pull the latest image URLs from Wikimedia Commons
// https://commons.wikimedia.org and map image colors to an xterm256 color
// palette.
package wikimg

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	// We define which image formats we support by importing
	// decoder packages
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var (
	// EndOfResults is returned by Next() when no more results are available
	EndOfResults = errors.New("end of results")

	// Cancelled may be returned by Next() and FirstColor() when the client
	// closes the Cancel channel on a Puller
	Cancelled = errors.New("cancelled")
)

const (
	// queryURL is the API we are querying
	queryURL = "https://commons.wikimedia.org/w/api.php"

	// apiMax is the max results we can request from the API at one time
	apiMax = 500

	// cancelCheckpoint
	cancelCheckpoint = 10000
)

// queryResp mirrors the JSON structure returned by queryURL, specifying only
// the info we're interested in.
type queryResp struct {

	// Continue contains strings we need to pass back into the API to
	// continue where we left off
	Continue struct {
		Continue   string
		AIContinue string
	}

	// Query contains the actual results
	Query struct {
		AllImages []struct {
			URL string
		}
	}
}

// Puller is an image puller that retrieves the most recent image URLs that
// have been uploaded to Wikimedia Commons https://commons.wikimedia.org
type Puller struct {
	// qr is the most recent response from the API
	qr *queryResp

	// i is the current index into qr.Query.AllImages
	i int

	// count is the total number of images we've collected
	count int

	// max is the maximum number of images we want to collect
	max int

	// Cancel is an optional channel. Setting this value on Puller
	// and closing the channel signals to the Puller that any
	// in process operations (i.e, retrieving an image or computing
	// its first color) should be cancelled. Any future
	// calls to Next() or FirstColor() will return a Cancelled
	// error.
	Cancel <-chan struct{}
}

// NewPuller creates a puller that can return at most max images
// when calls to Next() are made
func NewPuller(max int) *Puller {
	return &Puller{
		max: max,
	}
}

// Next returns the next most recent image URL. If no more results are
// available EndOfResults is returned as an error.
func (p *Puller) Next() (string, error) {
	// If we've exceeded that max we want to get, then stop
	if p.count >= p.max {
		return "", EndOfResults
	}

	// Ensure we haven't been cancelled yet
	select {
	case <-p.Cancel:
		// If p.Cancel has been closed, this will be triggered
		return "", Cancelled

	default:
		// Otherwise we'll just do nothing immediately
	}

	// If we're within the length of our current request,
	// return right away and increment our counters
	if p.qr != nil && p.i < len(p.qr.Query.AllImages) {
		img := p.qr.Query.AllImages[p.i].URL
		p.i++
		p.count++
		return img, nil
	}

	// Otherwise, we need to create a new request. Recreate our request params
	// and reset per-request counter.
	p.i = 0
	params := url.Values{}
	params.Set("action", "query")
	params.Set("format", "json")
	params.Set("list", "allimages")
	params.Set("aidir", "descending")
	params.Set("aisort", "timestamp")

	// 500 is the most allowed by the API per request, but we may want
	// less.
	if p.count+apiMax > p.max {
		params.Set("ailimit", strconv.Itoa(p.max-p.count))
	} else {
		params.Set("ailimit", strconv.Itoa(p.max))
	}

	// If we have a previous request with continue values, use them
	if p.qr != nil &&
		len(p.qr.Continue.Continue) > 0 &&
		len(p.qr.Continue.AIContinue) > 0 {
		params.Set("continue", p.qr.Continue.Continue)
		params.Set("aicontinue", p.qr.Continue.AIContinue)
	}

	// Call the wikimedia API
	resp, err := http.Get(queryURL + "?" + params.Encode())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the contents of the response as bytes
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the bytes into a struct
	p.qr = &queryResp{}
	err = json.Unmarshal(b, p.qr)
	if err != nil {
		return "", err
	}

	// If there's no more images, then return
	if len(p.qr.Query.AllImages) < 1 {
		return "", EndOfResults
	}

	// Return first value of the new request
	p.count++
	return p.qr.Query.AllImages[p.i].URL, nil
}

// FirstColor tries to return the first non-gray color in the image. A gray
// color is one that, when mapped to an xterm256 palette, has the same value
// for red, green and blue. Ok, we've defined "gray". What does "first" mean?
// We iterate through pixels starting with 0,0 (top left) and move to the
// bottom right one by one. In the worst case (a grayscale image), we iterate
// through every pixel, give up, and return the final pixel color even though
// it's gray. Both the xtermColor (an integer between 0-255) and a hex
// string (e.g., "#bb00cc") is returned.
func (p *Puller) FirstColor(imgURL string) (xtermColor int, hex string, err error) {
	// Create a request so we can use req.Cancel
	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return
	}

	// Set up cancellation pipeline, link request to puller
	req.Cancel = p.Cancel

	// Call the image server
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Decode into an object
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return
	}

	// Use our XTerm256 as a color.Palette so we can map the colors of the
	// image to our palette.
	pal := color.Palette(XTerm256)

	// Iterate through every pixel and try to find a color. If we don't
	// find a color (i.e., the image is grayscale) we'll default to the last
	// pixel in the image.
	rect := img.Bounds()
	i := 0
	for x := 0; x < rect.Dx(); x++ {
		for y := 0; y < rect.Dy(); y++ {

			// Check if p.Cancel has been closed once every cancelCheckpoint
			// iterations
			if i%cancelCheckpoint == 0 {
				select {

				case <-p.Cancel:
					err = Cancelled
					return
				default:

				}
			}

			// xtermColor is the index in the palette which this
			// actual color maps to. It is also (by design) the
			// xterm256 value that maps to this color.
			xtermColor = pal.Index(img.At(x, y))

			// Get the color.RGBA value for this color. Not great to do a type
			// assertion here but easiest way to get 8-bit values without bit
			// fiddling.
			rgba, ok := pal[xtermColor].(color.RGBA)
			if !ok {
				err = errors.New("can't assert to color.RGBA")
				return
			}

			// Compute the hex value of the color
			hex = fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)

			// If any of the RGB values differ, it's a color, so we can
			// stop.
			if !(rgba.R == rgba.G && rgba.G == rgba.B) {
				break
			}
		}
	}

	return
}
