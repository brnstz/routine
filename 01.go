package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/brnstz/routine/wikimg"
)

func main() {
	// Create a temporary dir to store our images
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	fmt.Println("images will be saved to: ", dir)

	p := wikimg.NewPuller(10000)

	i := 0
	for {
		imgURL, err := p.Next()
		if err == wikimg.EndofResults {
			break

		} else if err != nil {
			panic(err)
		}

		fmt.Println(imgURL, i)
		i++
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
