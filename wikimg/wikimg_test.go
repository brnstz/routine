package wikimg_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/brnstz/routine/wikimg"
)

func Example() {
	// Save 10 random images to a directory

	// Create a pull with max 10 results
	p := wikimg.NewPuller(10)

	// Create temp dir for storing the images
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		panic(err)
	}

	// Remove dir after test is complete
	defer os.RemoveAll(dir)

	for {
		// Get the next URL
		imgURL, err := p.Next()

		if err == wikimg.EndOfResults {
			// We've reached the end
			break
		} else if err != nil {
			// There's an unexpected error
			panic(err)
		}

		// Call GET on the image URL
		resp, err := http.Get(imgURL)
		if err != nil {
			panic(err)
		}

		// Open a temporary file
		fh, err := ioutil.TempFile(dir, "")
		if err != nil {
			// We need to close our HTTP response here too
			resp.Body.Close()
			panic(err)
		}

		// Copy GET results to file and close stuff
		_, err = io.Copy(fh, resp.Body)
		fh.Close()
		resp.Body.Close()
		if err != nil {
			panic(err)
		}
	}

	files, _ := ioutil.ReadDir(dir)
	fmt.Println(len(files))

	// Output: 10
}
