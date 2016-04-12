package main

import (
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"

	"github.com/brnstz/routine/wikimg"
)

func main() {

	http.HandleFunc("/colors", getImages)

	http.ListenAndServe(":8000", nil)
}

func getImages(w http.ResponseWriter, r *http.Request) {
	p := wikimg.NewPuller(1000)

	for {
		imgURL, err := p.Next()
		if err == wikimg.EndofResults {
			break

		} else if err != nil {
			log.Println(err)
			continue
		}

		go func() {
			c, err := getAvgColor(imgURL)
			if err != nil {
				log.Println(err)
				return
			}

			fmt.Fprintf(w, `<div style="background: #%x%x%x; width=100%%">#%x%x%x</div>`, c.R, c.G, c.B, c.R, c.G, c.B)

			f, ok := w.(http.Flusher)
			if ok {
				f.Flush()
			}
		}()
	}
}

func getAvgColor(imgURL string) (avgColor color.RGBA, err error) {
	var r, g, b, a uint32
	var pixels, tR, tG, tB, tA uint64

	// call the image server
	resp, err := http.Get(imgURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return
	}

	rect := img.Bounds()
	for x := 0; x < rect.Dx(); x++ {
		for y := 0; y < rect.Dy(); y++ {
			pixels++
			r, g, b, a = img.At(x, y).RGBA()
			tR += uint64(r)
			tG += uint64(g)
			tB += uint64(b)
			tA += uint64(a)
		}
	}
	avgColor = color.RGBA{
		uint8(tR / pixels),
		uint8(tG / pixels),
		uint8(tB / pixels),
		uint8(tA / pixels),
	}

	return
}
