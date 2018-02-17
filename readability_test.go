package readability

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

var urlWithAbsoluteImgPaths = "http://www.espn.com/nba/insider/story/_/id/22450965/drafting-nba-rising-stars-future-star-potential-ben-simmons-lonzo-ball-joel-embiid-more"
var urlWithRelativeImgPaths = "http://www.boogiejack.com/server_paths.html"

func TestExtract(t *testing.T) {
	opt := NewOption()
	opt.ImageRequestTimeout = 500
	c, err := Extract(urlWithAbsoluteImgPaths, opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, c.Title)
	assert.NotContains(t, c.Title, "\n")
	assert.NotEmpty(t, c.Description)
	assert.NotContains(t, c.Description, "\n")
	assert.Empty(t, c.Images) // empty since images are lazily-loaded

	c, err = Extract(urlWithRelativeImgPaths, opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, c.Title)
	assert.NotContains(t, c.Title, "\n")
	assert.NotEmpty(t, c.Description)
	assert.NotContains(t, c.Description, "\n")
	assert.NotEmpty(t, c.Images)
}

func TestExtractForImages(t *testing.T) {
	u := "http://www.orangesmile.com/travelguide/palermo/photo-gallery.htm"
	opt := NewOption()
	opt.IgnoreImageFormat = []string{"data:image/", ".svg", ".webp", ".gif"}
	opt.ImageRequestTimeout = 2000
	opt.CheckImageLoopCount = 20
	opt.MaxImageCount = 3
	opt.MinImageWidth = 300
	opt.MinImageHeight = 300
	c, _ := Extract(u, opt)
	assert.Equal(t, opt.MaxImageCount, len(c.Images))
}

func TestPattern(t *testing.T) {
	p := newPattern()
	assert.Empty(t, p.Video.FindString("http://WWW.ITUBE.COM"))
	assert.NotEmpty(t, p.Video.FindString("http://WWW.YOUTUBE.COM"))
	assert.NotEmpty(t, p.UnlikelyCandidates.FindString("My Comment"))
}

func TestClassWeight(t *testing.T) {
	html := `<div id="main-article" class="text blog">for positive class weight</div>
<a id="footer-link" class="btn" href="#">for negative class weight</a>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	s := doc.Find("div").First()
	assert.Equal(t, 50.0, classWeight(s, NewOption()))
	s = doc.Find("a").First()
	assert.Equal(t, -25.0, classWeight(s, NewOption()))
}

func TestLinkDensity(t *testing.T) {
	html := `<div>Speak blah blah!<a>123</a><a>4</a></div>`
	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))
	assert.Equal(t, 0.2, linkDensity(doc.Selection))
}

func TestAbsPath(t *testing.T) {
	// for absolute path
	url := "http://www.kakao.com/talk"
	in := url + "/img/a.jpg"
	out, err := absPath(in, url)
	assert.Nil(t, err)
	assert.Equal(t, in, out)

	// for relative path starting with "/"
	in = "/img/b.jpg"
	out, err = absPath(in, url)
	assert.Nil(t, err)
	assert.Equal(t, "http://www.kakao.com/img/b.jpg", out)

	// for relative path not starting with "/" and reqURL does not have subdirectories
	in = "img/b.jpg"
	out, err = absPath(in, "http://www.kakao.com")
	assert.Nil(t, err)
	assert.Equal(t, "http://www.kakao.com/img/b.jpg", out)

	// for relative path not starting with "/"
	url = "https://www.wto.org/english/tratop_e/envir_e/envir_req_e.htm"
	in = "../../../images/top_logo.gif"
	out, err = absPath(in, url)
	assert.Nil(t, err)
	assert.Equal(t, "https://www.wto.org/english/tratop_e/envir_e/../../../images/top_logo.gif", out)

	// for empty input path
	in = ""
	out, err = absPath(in, url)
	assert.Equal(t, "", out)
	assert.NotNil(t, err)

	// failing case - invalid input path
	url = "http://www.kakao.com"
	in = "fhsjkdfhjsdf#$%^#&^"
	_, err = absPath(in, url)
	assert.NotNil(t, err)

	// failing case - invalid requestURL string
	url = "yirqywi8r4o"
	in = "/a.jpg"
	_, err = absPath(in, url)
	assert.NotNil(t, err)
}

func TestAbsPathWithoutScheme(t *testing.T) {
	url := "https://brunch.co.kr/@julieted17/19"
	in := "//t1.daumcdn.net/brunch/static/icon/favicon/favicon64_150520.ico"
	out, err := absPath(in, url)
	assert.Nil(t, err)
	assert.Equal(t, "https:"+in, out)
}

func TestDescriptionTimeout(t *testing.T) {
	url := "https://tools.ietf.org/rfc/"
	opt := NewOption()
	opt.DescriptionExtractionTimeout = 50
	c, err := Extract(url, opt)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Empty(t, c.Description)
	assert.Empty(t, c.Images)
}

func TestAuthor(t *testing.T) {
	// <span class='author'>Jonathan Givony and Mike Schmitz</span>
	doc, _ := goquery.NewDocument(urlWithAbsoluteImgPaths)
	assert.Equal(t, "Jonathan Givony and Mike Schmitz", author(doc))

	// <meta name="dc.creator" content="Finch" />
	html := `<head><meta name="dc.creator" content="Finch" /></head>`
	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(html))
	assert.Equal(t, "Finch", author(doc))

	// <meta name="author" content="philip" />
	html = `<head><meta name="author" content="philip" /></head>`
	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(html))
	assert.Equal(t, "philip", author(doc))

	// <a rel="author" href="http://dbanksdesign.com">Danny Banks (rel)</a>
	html = `<a rel="author" href="http://dbanksdesign.com">Danny Banks (rel)</a>`
	doc, _ = goquery.NewDocumentFromReader(strings.NewReader(html))
	assert.Equal(t, "Danny Banks (rel)", author(doc))
}
