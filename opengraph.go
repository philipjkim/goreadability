package readability

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

// OpenGraph contains opengraph meta values.
type OpenGraph struct {
	Title       string `json:"og:title,omitempty"`
	Description string `json:"og:description,omitempty"`
	ImageURL    string `json:"og:image,omitempty"`
}

// Set sets value to the key-related field.
func (og *OpenGraph) Set(key string, val string, urlStr string) error {
	switch key {
	case "og:title":
		og.Title = val
	case "og:description":
		og.Description = val
	case "og:image":
		var err error
		og.ImageURL, err = absPath(val, urlStr)
		if err != nil {
			logger.Printf("OpenGraph.Set failed: %v", err)
		}
	default:
		return fmt.Errorf("Invalid key for OpenGraph.Set: %v", key)
	}
	return nil
}

// IsEmpty returns true if all fields of og are empty.
func (og OpenGraph) IsEmpty() bool {
	return og.Title == "" &&
		og.Description == "" &&
		og.ImageURL == ""
}

var metaProps = []string{
	"og:title",
	"og:description",
	"og:image",
}

func getContentFromOpenGraph(doc *goquery.Document, reqURL string) (*OpenGraph, error) {
	og := OpenGraph{}
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {
		k, ke := s.Attr("property")
		if !ke {
			k, ke = s.Attr("name")
		}
		if !ke {
			k, ke = s.Attr("itemprop")
		}
		v, ve := s.Attr("content")

		if !ke || !ve {
			return
		}

		for _, key := range metaProps {
			if k == key {
				og.Set(k, v, reqURL)
			}
		}
	})
	logger.Printf("OpenGraph: %v\n", og)
	return &og, nil
}
