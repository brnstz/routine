package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

func main() {
	// Create a temporary dir to store our images
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	fmt.Println("images will be saved to: ", dir)

	// get all the images
	err = getImages(dir, 15)
	if err != nil {
		panic(err)
	}
}

// getImage downloads the contents of imageURL to file in dir
func getImage(dir, imageURL string) error {
	// create a temporary file for saving the image
	file, err := ioutil.TempFile(dir, "")
	if err != nil {
		return err
	}
	defer file.Close()

	// call the image server
	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// copy from the HTTP response to our temporary file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	// success!
	return nil
}

// getImages calls queryURL, parses the response and downloads all
// image URLs in the response to dir.
func getImages(dir string, max int) error {
	// count is the total results we've received already
	var count int

	// continue_ and aicontninue are saved from the previous request
	// to pass back to the API to continue
	var continue_, aicontinue string

	for {
		// If we've exceeded that max we want to get, then stop
		if count >= max {
			break
		}

		// Recreate params on each request. We're getting the most
		// recently uploaded images.
		params := url.Values{}
		params.Set("action", "query")
		params.Set("format", "json")
		params.Set("list", "allimages")
		params.Set("aidir", "descending")
		params.Set("aisort", "timestamp")

		// 500 is the most allowed by the API per request, but we may want
		// less.
		if count+apiMax > max {
			params.Set("ailimit", strconv.Itoa(max-count))
		} else {
			params.Set("ailimit", strconv.Itoa(max-count))
		}

		if len(continue_) > 0 && len(aicontinue) > 0 {
			params.Set("continue", continue_)
			params.Set("aicontinue", aicontinue)
		}

		// call the wikimedia API
		resp, err := http.Get(queryURL + "?" + params.Encode())
		if err != nil {
			return err
		}

		// read the contents of the response as bytes
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		resp.Body.Close()

		// parse the bytes into a struct
		qr := queryResp{}
		err = json.Unmarshal(b, &qr)
		if err != nil {
			return err
		}

		// If there's no more images, then break
		if len(qr.Query.AllImages) < 1 {
			break
		}

		// iterate over all image URLs
		for _, img := range qr.Query.AllImages {
			count++
			continue_ = qr.Continue.Continue
			aicontinue = qr.Continue.AIContinue

			log.Println(img.URL, count)

			// go get that image!
			//	go getImage(dir, img.URL)
		}
	}

	// success!
	return nil
}

const (
	queryURL = "https://commons.wikimedia.org/w/api.php"
	apiMax   = 500
)

// queryResp mirrors the JSON structure returned by queryURL, specifying only
// the info we're interested in.
type queryResp struct {
	// To get the next page of results, we pass these values back to the
	// API
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
