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
	err = getImages(dir, 501)
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
	var count int

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
	Query struct {
		AllImages []struct {
			URL string
		}
	}
}
