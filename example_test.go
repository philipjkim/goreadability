package readability_test

import (
	"log"

	"github.com/philipjkim/goreadability"
)

func Example() {
	// URL to extract contents (title, description, images, ...)
	url := "https://en.wikipedia.org/wiki/Lego"

	// Default option
	opt := readability.NewOption()

	// You can modify some option values if needed.
	opt.ImageFetchTimeout = 3000 // ms

	content, err := readability.Extract(url, opt)
	if err != nil {
		log.Fatal(err)
	}

	log.Println(content.Title)
	log.Println(content.Description)
	log.Println(content.Images)
}
