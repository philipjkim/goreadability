goreadability
=============

[![GoDoc](https://godoc.org/github.com/philipjkim/goreadability?status.svg)](https://godoc.org/github.com/philipjkim/goreadability) [![Build Status](https://travis-ci.org/philipjkim/goreadability.png)](https://travis-ci.org/philipjkim/goreadability)

goreadability is a tool for extracting the primary readable content of a
webpage. It is a Go port of arc90's readability project, 
based on [ruby-readability](https://github.com/cantino/ruby-readability). 


Install
-------

    go get github.com/philipjkim/goreadability


Example
-------

```go
r, err := readability.ExtractFromResponse("http://m.twins.mlb.com/news/article/172850240/twins-impressed-by-byung-ho-parks-home-run")
if err != nil {
    // Something went wrong.
    panic(err)
}

fmt.Println(r)
```


Options
-------

TODO


Command Line Tool
-----------------

TODO


Related Projects
----------------

* [ruby-readability](https://github.com/cantino/ruby-readability) is the base of this project.


Potential Issues
----------------

TODO


License
-------

This code is under the Apache License 2.0. See <http://www.apache.org/licenses/LICENSE-2.0>.


[![Bitdeli Badge](https://d2weczhvl823v0.cloudfront.net/philipjkim/goreadability/trend.png)](https://bitdeli.com/free "Bitdeli Badge")

