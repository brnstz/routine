package wikimg

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

var EndofResults = errors.New("end of results")

const (
	queryURL = "https://commons.wikimedia.org/w/api.php"
	apiMax   = 500
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

	// The actual results
	Query struct {
		AllImages []struct {
			URL string
		}
	}
}

type Puller struct {
	// qr is the most recent response from the API
	qr *queryResp

	// i is the current index into qr.Query.AllImages
	i int

	// count is the total number of images we've collected
	count int

	// max is the maximum number of images we want to collect
	max int

	// err is the most recent error which can be returned by ImageURL()
	err error
}

// NewPuller creates an image puller that pulls up to max of the most
// recent image URLs that have been uploaded to Wikimedia Commons
// https://commons.wikimedia.org
func NewPuller(max int) *Puller {
	return &Puller{
		max: max,
	}
}

func (p *Puller) Next() (string, error) {
	// If we've exceeded that max we want to get, then stop
	if p.count > p.max {
		return "", EndofResults
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

	// call the wikimedia API
	resp, err := http.Get(queryURL + "?" + params.Encode())
	if err != nil {
		return "", err
	}

	// read the contents of the response as bytes
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	// parse the bytes into a struct
	p.qr = &queryResp{}
	err = json.Unmarshal(b, p.qr)
	if err != nil {
		return "", err
	}

	// If there's no more images, then return false
	if len(p.qr.Query.AllImages) < 1 {
		return "", EndofResults
	}

	// Return first value of the new request
	p.count++
	return p.qr.Query.AllImages[p.i].URL, nil
}
