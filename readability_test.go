package readability

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

var urlWithAbsoluteImgPaths = "http://m.twins.mlb.com/news/article/172850240/twins-impressed-by-byung-ho-parks-home-run"
var urlWithRelativeImgPaths = "http://weplanner.co.kr/?webid=160815854"

func TestExtract(t *testing.T) {
	opt := NewOption()
	c, err := Extract(urlWithAbsoluteImgPaths, opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, c.Title)
	assert.NotContains(t, c.Title, "\n")
	assert.NotEmpty(t, c.Description)
	assert.NotContains(t, c.Description, "\n")
	assert.NotEmpty(t, c.Images)

	c, err = Extract(urlWithRelativeImgPaths, opt)
	assert.Nil(t, err)
	assert.NotEmpty(t, c.Title)
	assert.NotContains(t, c.Title, "\n")
	assert.NotEmpty(t, c.Description)
	assert.NotContains(t, c.Description, "\n")
	assert.NotEmpty(t, c.Images)
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
