goreadability
=============

[![GoDoc](https://godoc.org/github.com/philipjkim/goreadability?status.svg)](https://godoc.org/github.com/philipjkim/goreadability) [![Go Report Card](https://goreportcard.com/badge/github.com/philipjkim/goreadability)](https://goreportcard.com/report/github.com/philipjkim/goreadability) [![Build Status](https://travis-ci.org/philipjkim/goreadability.svg)](https://travis-ci.org/philipjkim/goreadability)

goreadability is a tool for extracting the primary readable content of a
webpage. It is a Go port of arc90's readability project, 
based on [ruby-readability](https://github.com/cantino/ruby-readability). 


Install
-------

    go get github.com/philipjkim/goreadability


Example
-------

```go
// URL to extract contents (title, description, images, ...)
url := "https://en.wikipedia.org/wiki/Lego"

// Default option
opt := readability.NewOption()

// You can modify some option values if needed.
opt.ImageRequestTimeout = 3000 // ms

content, err := readability.Extract(url, opt)
if err != nil {
    log.Fatal(err)
}

log.Println(content.Title)
log.Println(content.Description)
log.Println(content.Images)
```


Command Line Tool
-----------------

TODO


Related Projects
----------------

* [ruby-readability](https://github.com/cantino/ruby-readability) is the base of this project.
* [fastimage](https://github.com/rubenfonseca/fastimage) finds the type and/or size of a remote image given its uri, by fetching as little as needed.


Potential Issues
----------------

TODO


License
-------

[MIT](LICENSE)
