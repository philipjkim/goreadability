// Port of arc90's readability project to Go

package readability

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/PuerkitoBio/goquery"
	"github.com/philipjkim/fastimage"
)

// Image contains URL and Size (width and height in pixel).
type Image struct {
	URL  string
	Size *fastimage.ImageSize
}

func (i Image) String() string {
	return fmt.Sprintf("{URL: %v, Size: %vx%v}", i.URL, i.Size.Width, i.Size.Height)
}

// Option contains variety of options for extracting page content and images.
type Option struct {
	// RetryLength is minimum length for a page description.
	// It will retry to extract page description with more liberal rule
	// if extracted description length is less than this value.
	RetryLength int

	// MinTextLength is minimum length of an inner text for a tag.
	// If a tag has short inner text (length is less than MinTextLength),
	// the text will be discarded from the page description candidates.
	MinTextLength int

	// RemoveUnlikelyCandidates is a flag whether to remove some tags
	// if they are considered relatively unimportant.
	RemoveUnlikelyCandidates bool

	// WeightClasses is a flag whether to give more/less weight to some tags
	// if they contain some positive/negative words in id/class value.
	WeightClasses bool

	// CleanConditionally is a flag whether to remove some tags
	// using various rules in conditionalCleanReason().
	CleanConditionally bool

	// RemoveEmptyNodes is a flag whether to remove some tags which have empty inner text.
	RemoveEmptyNodes bool

	// MinImageWidth is the minimum width (pixel) for choosing images.
	MinImageWidth uint32

	// MinImageHeight is the minimum height (pixel) for choosing images.
	MinImageHeight uint32

	// MaxImageCount is the maximum number of images for a web page.
	MaxImageCount int

	// CheckImageLoopCount is the number of images
	// for parallel requests to fetch the image size.
	// For example, if this value is set to 10,
	// the first 10 img src URLs without width/height attributes
	// will be requested over network.
	// (img tags with both width/height attributes (pixels in int) are not conunted,
	// since they are not requested over network to get image size.)
	CheckImageLoopCount uint

	// ImageRequestTimeout is timeout(ms) for a single image request.
	ImageRequestTimeout uint

	// IgnoreImageFormat is an array of strings for ignoring some images.
	// If an image URL contains at least one of strings in this array, the image will be ignored.
	IgnoreImageFormat []string

	// DescriptionAsPlainText is a flag whether to strip all tags in a description value.
	DescriptionAsPlainText bool

	// DescriptionExtractionTimeout is timeout(ms) for extracting description for a page.
	DescriptionExtractionTimeout uint
}

// NewOption returns the default option.
func NewOption() *Option {
	return &Option{
		RetryLength:                  250,
		MinTextLength:                25,
		RemoveUnlikelyCandidates:     true,
		WeightClasses:                true,
		CleanConditionally:           true,
		RemoveEmptyNodes:             true,
		MinImageWidth:                200,
		MinImageHeight:               100,
		MaxImageCount:                3,
		CheckImageLoopCount:          10,
		ImageRequestTimeout:          1000,
		IgnoreImageFormat:            []string{"data:image/", ".svg", ".webp"},
		DescriptionAsPlainText:       true,
		DescriptionExtractionTimeout: 500,
	}
}

func copyOption(o *Option) *Option {
	return &Option{
		RetryLength:                  o.RetryLength,
		MinTextLength:                o.MinTextLength,
		RemoveUnlikelyCandidates:     o.RemoveUnlikelyCandidates,
		WeightClasses:                o.WeightClasses,
		CleanConditionally:           o.CleanConditionally,
		RemoveEmptyNodes:             o.RemoveEmptyNodes,
		MinImageWidth:                o.MinImageWidth,
		MinImageHeight:               o.MinImageHeight,
		MaxImageCount:                o.MaxImageCount,
		CheckImageLoopCount:          o.CheckImageLoopCount,
		ImageRequestTimeout:          o.ImageRequestTimeout,
		IgnoreImageFormat:            o.IgnoreImageFormat,
		DescriptionAsPlainText:       o.DescriptionAsPlainText,
		DescriptionExtractionTimeout: o.DescriptionExtractionTimeout,
	}
}

type pattern struct {
	UnlikelyCandidates   *regexp.Regexp
	OKMaybeItsACandidate *regexp.Regexp
	Positive             *regexp.Regexp
	Negative             *regexp.Regexp
	DivToPElements       *regexp.Regexp
	ReplaceBrs           *regexp.Regexp
	ReplaceFonts         *regexp.Regexp
	Normalize            *regexp.Regexp
	KillBreaks           *regexp.Regexp
	Video                *regexp.Regexp
	Tag                  *regexp.Regexp
	Trimmable            *regexp.Regexp
}

func newPattern() *pattern {
	uc := regexp.MustCompile("(?i)combx|comment|community|disqus|extra|foot|header|menu|remark|rss|shoutbox|sidebar|sponsor|ad-break|agegate|pagination|pager|popup")
	mc := regexp.MustCompile("(?i)and|article|body|column|main|shadow")
	pos := regexp.MustCompile("(?i)article|body|content|entry|hentry|main|page|pagination|post|text|blog|story")
	neg := regexp.MustCompile("(?i)combx|comment|com-|contact|foot|footer|footnote|masthead|media|meta|outbrain|promo|related|scroll|shoutbox|sidebar|sponsor|shopping|tags|tool|widget")
	dtp := regexp.MustCompile("(?i)<(a|blockquote|dl|div|img|ol|p|pre|table|ul)")
	rb := regexp.MustCompile("(?i)(<br[^>]*>[ \n\r\t]*){2,}")
	rf := regexp.MustCompile("(?i)<(\\/?)font[^>]*>")
	nm := regexp.MustCompile("\\s{2,}")
	kb := regexp.MustCompile("(<br\\s*\\/?>(\\s|&nbsp;?)*){1,}")
	vid := regexp.MustCompile("(?i)http:\\/\\/(www\\.)?(youtube|vimeo)\\.com")
	tag := regexp.MustCompile("<.*?>")
	tr := regexp.MustCompile("[\r\n\t ]+")
	return &pattern{
		UnlikelyCandidates:   uc,
		OKMaybeItsACandidate: mc,
		Positive:             pos,
		Negative:             neg,
		DivToPElements:       dtp,
		ReplaceBrs:           rb,
		ReplaceFonts:         rf,
		Normalize:            nm,
		KillBreaks:           kb,
		Video:                vid,
		Tag:                  tag,
		Trimmable:            tr,
	}
}

var patterns = newPattern()

// Content contains primary readable content of a webpage.
type Content struct {
	Title       string
	Description string
	Author      string
	Images      []Image
}

// Extract requests to reqURL then returns contents extracted from the response.
func Extract(reqURL string, opt *Option) (*Content, error) {
	doc, err := goquery.NewDocument(reqURL)
	if err != nil {
		return nil, err
	}
	return ExtractFromDocument(doc, reqURL, opt)
}

// ExtractFromDocument returns Content when extraction succeeds, otherwise error.
// reqURL is required for converting relative image paths to absolute.
//
// If you already have *goquery.Document after requesting HTTP, use this function,
// otherwise use Extract(reqURL, opt).
func ExtractFromDocument(doc *goquery.Document, reqURL string, opt *Option) (*Content, error) {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	return &Content{
		Title:       title,
		Description: description(doc, opt),
		Author:      author(doc),
		Images:      images(doc, reqURL, opt),
	}, nil
}

func description(doc *goquery.Document, opt *Option) string {
	candidates, err := prepareCandidates(doc, opt)
	if err != nil {
		return ""
	}
	article, err := getArticle(candidates)
	if err != nil {
		return ""
	}
	cleanedArticle := sanitize(article, candidates, opt)
	if opt.DescriptionAsPlainText {
		cleanedArticle = patterns.Tag.ReplaceAllString(cleanedArticle, " ")
		cleanedArticle = patterns.Trimmable.ReplaceAllString(cleanedArticle, " ")

	}
	if len(cleanedArticle) < opt.RetryLength {
		newOpts := copyOption(opt)
		if newOpts.RemoveUnlikelyCandidates {
			newOpts.RemoveUnlikelyCandidates = false
		} else if newOpts.WeightClasses {
			newOpts.WeightClasses = false
		} else if newOpts.CleanConditionally {
			newOpts.CleanConditionally = false
		} else {
			return cleanedArticle
		}
		return description(doc, newOpts)
	}

	return cleanedArticle
}

func prepareCandidates(doc *goquery.Document, opt *Option) (*candidates, error) {
	doc.Find("style, script").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	err := removeUnlikelyCandidates(doc, opt)
	if err != nil {
		return nil, err
	}
	err = transformMisusedDivsIntoP(doc, opt)
	if err != nil {
		return nil, err
	}

	return getCandidates(doc, opt)
}

func getArticle(candidates *candidates) (*goquery.Document, error) {
	if candidates == nil || len(candidates.List) == 0 {
		return nil, errors.New("Empty candidates")
	}
	bestCandidate := candidates.List[0]
	siblingScoreThreshold := math.Max(10.0, bestCandidate.Score*0.2)
	output, _ := goquery.NewDocumentFromReader(strings.NewReader("<div></div>"))
	re := regexp.MustCompile("\\.( |$)")
	bestCandidate.Node.Parent().Children().Each(func(i int, s *goquery.Selection) {
		sel := newMySelection(s)
		append := false
		if sel.HTML() == bestCandidate.Node.HTML() {
			append = true
		}
		if candidates.Map[sel.HTML()].Score >= siblingScoreThreshold {
			append = true
		}

		if goquery.NodeName(s) == "p" {
			ld := linkDensity(s)
			text := s.Text()
			length := len(text)

			if length > 80 && ld < 0.25 {
				append = true
			} else if length < 80 && ld == 0 && re.FindString(text) != "" {
				append = true
			}
		}

		if append {
			sCopy := s.Clone()
			if goquery.NodeName(s) != "div" && goquery.NodeName(s) != "p" {
				sCopy.Get(0).Data = "div"
			}
			output.AppendSelection(sCopy)
		}
	})
	return output, nil
}

func sanitize(doc *goquery.Document, candidates *candidates, opt *Option) string {
	doc.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		if classWeight(s, opt) < 0 || linkDensity(s) > 0.33 {
			s.Remove()
		}
	})
	doc.Find("form, object, iframe, embed").Each(func(i int, s *goquery.Selection) {
		s.Remove()
	})

	if opt.RemoveEmptyNodes {
		doc.Find("p").Each(func(i int, s *goquery.Selection) {
			if strings.TrimSpace(s.Text()) == "" {
				s.Remove()
			}
		})
	}

	cleanConditionally(doc, candidates, "table, ul, div", opt)

	whitelist := map[string]bool{"div": true, "p": true}
	st := []string{"br", "hr", "h1", "h2", "h3", "h4", "h5", "h6", "dl", "dd",
		"ol", "li", "ul", "address", "blockquote", "center"}
	spacey := map[string]bool{}
	for _, tag := range st {
		spacey[tag] = true
	}

	doc.Find("*").Each(func(i int, s *goquery.Selection) {
		tagName := goquery.NodeName(s)
		// If element is in whitelist, delete all its attributes
		if whitelist[tagName] {
			s.Nodes[0].Attr = []html.Attribute{}
		} else {
			// If element is root, replace the node as a text node
			if s.Parent() == nil {
				s.ReplaceWithHtml(s.Text())
			} else {
				if spacey[tagName] {
					s.ReplaceWithHtml(" " + s.Text() + " ")
				} else {
					s.ReplaceWithHtml(s.Text())
				}
			}
		}
	})

	re := regexp.MustCompile("[\r\n\f]+")
	html, _ := doc.Html()
	return re.ReplaceAllString(html, "\n")
}

func cleanConditionally(doc *goquery.Document, candidates *candidates, selector string, opt *Option) {
	if !opt.CleanConditionally {
		return
	}

	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		sel := newMySelection(s)
		weight := classWeight(s, opt)
		score := candidates.Map[sel.HTML()].Score
		tagName := goquery.NodeName(s)

		if weight+score < 0 {
			s.Remove()
		} else if strings.Count(s.Text(), ",") < 11 {
			counts := map[string]int{}
			for _, tag := range []string{"p", "img", "li", "a", "embed", "input"} {
				counts[tag] = s.Find(tag).Length()
				counts["li"] -= 100
				// For every img under a noscript tag discount one from the count to avoid double counting
				counts["img"] -= s.Find("noscript").Find("img").Length()
				cl := len(strings.TrimSpace(s.Text()))
				ld := linkDensity(s)
				reason := conditionalCleanReason(tagName, counts, cl, opt, weight, ld)
				if reason != "" {
					s.Remove()
				}
			}
		}
	})
}

func conditionalCleanReason(tagName string, counts map[string]int,
	cl int, opt *Option, weight float64, ld float64) string {
	if counts["img"] > counts["p"] && counts["img"] > 1 {
		return "too many images"
	} else if counts["li"] > counts["p"] && tagName != "ul" && tagName != "ok" {
		return "more <li>s than <p>s"
	} else if counts["input"]*3 > counts["p"] {
		return "<p>s less than 3 * <inputs>s"
	} else if cl < opt.MinTextLength && counts["img"] != 1 {
		return "too short content length without a single image"
	} else if (weight < 25 && ld > 0.2) || (weight >= 25 && ld > 0.5) {
		return "too many links for its weight"
	} else if (counts["embed"] == 1 && cl < 75) || counts["embed"] > 1 {
		return "<embed>s with too short content length, or too many <embed>s"
	} else {
		return ""
	}
}

func removeUnlikelyCandidates(doc *goquery.Document, opt *Option) error {
	if !opt.RemoveUnlikelyCandidates {
		return nil
	}

	ch := make(chan error)
	quit := false

	go func() {
		sel := doc.Find("*")
		if quit {
			return
		}
		sel.EachWithBreak(func(i int, s *goquery.Selection) bool {
			if quit {
				return false
			}
			cls, _ := s.Attr("class")
			id, _ := s.Attr("id")
			str := cls + id
			if patterns.UnlikelyCandidates.FindString(str) != "" &&
				patterns.OKMaybeItsACandidate.FindString(str) == "" &&
				goquery.NodeName(s) != "html" &&
				goquery.NodeName(s) != "body" {
				s.Remove()
			}
			return true
		})
		ch <- nil
		return
	}()

	timeout := time.After(time.Duration(opt.DescriptionExtractionTimeout) * time.Millisecond)
	select {
	case err := <-ch:
		return err
	case <-timeout:
		quit = true
		return errors.New("readability.removeUnlikelyCandidates timed out")
	}
}

func transformMisusedDivsIntoP(doc *goquery.Document, opt *Option) error {
	ch := make(chan error)
	quit := false

	go func() {
		sel := doc.Find("*")
		if quit {
			return
		}
		sel.EachWithBreak(func(i int, s *goquery.Selection) bool {
			if quit {
				return false
			}
			if goquery.NodeName(s) == "div" {
				innerHTML, _ := s.Html()
				if patterns.DivToPElements.FindString(innerHTML) == "" {
					s.Get(0).Data = "p"
				}
			}
			return true
		})
		ch <- nil
		return
	}()

	timeout := time.After(time.Duration(opt.DescriptionExtractionTimeout) * time.Millisecond)
	select {
	case err := <-ch:
		return err
	case <-timeout:
		quit = true
		return errors.New("readability.transformMisusedDivsIntoP timed out")
	}
}

func getCandidates(doc *goquery.Document, opt *Option) (*candidates, error) {
	ch := make(chan *candidates)
	quit := false

	go func() {
		cMap := map[string]candidate{}
		doc.Find("p, td").EachWithBreak(func(i int, s *goquery.Selection) bool {
			if quit {
				return false
			}
			parent := s.Parent()
			var grandParent *goquery.Selection
			if parent == nil {
				grandParent = nil
			} else {
				grandParent = parent.Parent()
			}
			innerText := s.Text()

			if len(innerText) < opt.MinTextLength {
				return true
			}

			score := 1.0
			score += float64(len(strings.Split(innerText, ",")))
			score += math.Min((float64(len(innerText)) / 100.0), 3.0)

			psel := newMySelection(parent)
			if _, ok := cMap[psel.HTML()]; !ok {
				cMap[psel.HTML()] = candidate{Node: psel, Score: scoreNode(parent, opt) + score}
			}

			if grandParent != nil {
				gsel := newMySelection(grandParent)
				if _, ok := cMap[gsel.HTML()]; !ok {
					cMap[gsel.HTML()] = candidate{
						Node:  gsel,
						Score: scoreNode(grandParent, opt) + (score / 2.0),
					}
				}
			}
			return true
		})

		// Scale the final candidates score based on link density.
		// Good content should have a relatively small link density (5% or less)
		// and be mostly unaffected by this operation.
		for k, v := range cMap {
			cMap[k] = candidate{Node: v.Node, Score: v.Score * (1 - linkDensity(v.Node.Selection))}
		}
		ch <- &candidates{Map: cMap, List: sortCandidates(cMap)}
		return
	}()

	timeout := time.After(time.Duration(opt.DescriptionExtractionTimeout) * time.Millisecond)
	for {
		select {
		case result := <-ch:
			quit = true
			return result, nil
		case <-timeout:
			quit = true
			return nil, errors.New("readability.getCandidates timed out")
		}
	}
}

var elemScores = map[string]float64{
	"div":        5,
	"blockquote": 3,
	"form":       -3,
	"th":         -5,
}

func scoreNode(s *goquery.Selection, opt *Option) float64 {
	score := classWeight(s, opt)
	es := elemScores[s.Get(0).Data]
	score += es
	return score
}

func classWeight(s *goquery.Selection, opt *Option) float64 {
	weight := 0.0

	if !opt.WeightClasses {
		return weight
	}

	if c, _ := s.Attr("class"); c != "" {
		if patterns.Negative.FindString(c) != "" {
			weight -= 25.0
		}
		if patterns.Positive.FindString(c) != "" {
			weight += 25.0
		}
	}
	if i, _ := s.Attr("id"); i != "" {
		if patterns.Negative.FindString(i) != "" {
			weight -= 25.0
		}
		if patterns.Positive.FindString(i) != "" {
			weight += 25.0
		}
	}
	return weight
}

func linkDensity(s *goquery.Selection) float64 {
	linkTexts := s.Find("a").Map(func(i int, s *goquery.Selection) string {
		return s.Text()
	})
	linkLen := float64(len(strings.Join(linkTexts, "")))
	textLen := float64(len(s.Text()))
	return linkLen / textLen
}

type mySelection struct {
	*goquery.Selection
}

func newMySelection(s *goquery.Selection) *mySelection {
	return &mySelection{s}
}

func (s *mySelection) HTML() string {
	html, _ := s.Html()
	return html
}

func (s *mySelection) String() string {
	if s == nil {
		return "(nil)"
	}
	return fmt.Sprintf("%v#%v.%v",
		goquery.NodeName(s.Selection),
		s.AttrOr("id", "(undefined)"),
		s.AttrOr("class", "(undefined)"))
}

type candidate struct {
	Node  *mySelection
	Score float64
}

func (c candidate) String() string {
	if c.Node == nil {
		return ""
	}
	return fmt.Sprintf("%v(%v)", c.Node, c.Score)
}

type candidateList []candidate

func (c candidateList) Len() int {
	return len(c)
}
func (c candidateList) Less(i, j int) bool {
	return c[i].Score < c[j].Score
}
func (c candidateList) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

type candidates struct {
	Map  map[string]candidate
	List candidateList
}

func sortCandidates(candidates map[string]candidate) candidateList {
	cl := make(candidateList, len(candidates))
	i := 0
	for _, v := range candidates {
		cl[i] = v
		i++
	}
	sort.Sort(sort.Reverse(cl))
	return cl
}

func images(doc *goquery.Document, reqURL string, opt *Option) []Image {
	ch := make(chan *Image)
	defer close(ch)

	imgs := []Image{}
	loopCnt := uint(0)
	doc.Find("img").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if loopCnt >= opt.CheckImageLoopCount {
			return false
		}
		src, err := absPath(s.AttrOr("src", s.AttrOr("data-original", "")), reqURL)
		if err != nil {
			return true
		}
		if !isSupportedImage(src, opt) {
			return true
		}

		w, _ := strconv.Atoi(s.AttrOr("width", "0"))
		h, _ := strconv.Atoi(s.AttrOr("height", "0"))
		if isVerbose() {
			fmt.Printf("loopCnt: %v, src: %v, w: %v, h: %v\n", loopCnt, src, w, h)
		}

		go func(lc *uint) {
			defer func() {
				if err := recover(); err != nil {
					if isVerbose() {
						fmt.Printf("[readability] checkImageSize failed for %v: %v\n",
							src, err)
					}
				}
			}()

			ch <- checkImageSize(src, w, h, opt, lc)
		}(&loopCnt)

		return true
	})

	timeout := time.After(time.Duration(opt.ImageRequestTimeout+50) * time.Millisecond)
	for {
		select {
		case result := <-ch:
			if result.Size != nil &&
				result.Size.Width >= opt.MinImageWidth &&
				result.Size.Height >= opt.MinImageHeight {
				imgs = append(imgs, *result)
			}
			if len(imgs) >= opt.MaxImageCount {
				return imgs
			}
		case <-timeout:
			if isVerbose() {
				fmt.Printf("[readability] checkImageSize timed out: reqURL: %s\n", reqURL)
			}
			return imgs
		}
	}
}

func isSupportedImage(src string, opt *Option) bool {
	for _, ext := range opt.IgnoreImageFormat {
		if strings.Contains(src, ext) {
			return false
		}
	}
	return true
}

func checkImageSize(src string, widthFromAttr, heightFromAttr int, opt *Option, loopCnt *uint) *Image {
	width, height := widthFromAttr, heightFromAttr
	if width == 0 || height == 0 {
		*loopCnt++
		_, size, err := fastimage.DetectImageTypeWithTimeout2(src, opt.ImageRequestTimeout)
		if isVerbose() {
			fmt.Printf("[req] loopCnt: %v, src: %v, err: %v, size: %v\n",
				*loopCnt, src, err, size)
		}
		if err != nil {
			return &Image{}
		}
		if size != nil {
			width, height = int(size.Width), int(size.Height)
		}
	}
	return &Image{
		URL:  src,
		Size: &fastimage.ImageSize{Width: uint32(width), Height: uint32(height)},
	}
}

func author(doc *goquery.Document) string {
	var author string
	var found bool

	// <meta name="dc.creator" content="Finch - http://www.getfinch.com" />
	// <meta name="author" content="Soo Philip Jason Kim" />
	doc.Find("meta").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.AttrOr("name", "") == "dc.creator" {
			author, found = s.Attr("content")
			if found {
				return false
			}
		}
		if s.AttrOr("name", "") == "author" {
			author, found = s.Attr("content")
			if found {
				return false
			}
		}
		return true
	})
	if author != "" {
		return author
	}

	// <span class="author"><span class="faded">By</span> Rhett Bollinger</span>
	doc.Find("span.author").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.Text() != "" {
			author = strings.TrimSpace(s.Text())
			return false
		}
		return true
	})
	if author != "" {
		return author
	}

	// <a rel="author" href="http://dbanksdesign.com">Danny Banks (rel)</a>
	doc.Find("a").EachWithBreak(func(i int, s *goquery.Selection) bool {
		if s.AttrOr("rel", "") == "author" {
			author = strings.TrimSpace(s.Text())
			return false
		}
		return true
	})
	return author
}

func absPath(in string, reqURLStr string) (out string, err error) {
	if strings.TrimSpace(in) == "" {
		return "", errors.New("Empty input string for absPath")
	}

	inURL, err := url.Parse(in)
	if err != nil {
		return "", err
	}

	if inURL.IsAbs() {
		return in, nil
	}

	reqURL, err := url.Parse(reqURLStr)
	if err != nil {
		return "", err
	}
	if !isValidURLStr(reqURLStr) {
		return "", fmt.Errorf("url %v has invalid scheme", reqURLStr)
	}

	if strings.HasPrefix(in, "//") {
		return reqURL.Scheme + ":" + in, nil
	}
	if strings.HasPrefix(in, "/") {
		return reqURL.Scheme + "://" + reqURL.Host + in, nil
	}

	var result string
	sPos := strings.LastIndex(reqURLStr, "/")
	if sPos < 8 {
		result = reqURLStr + "/" + in
	} else {
		result = reqURLStr[:sPos+1] + in
	}
	_, err = url.Parse(result)
	if err != nil {
		return "", err
	}
	return result, nil
}

func isValidURLStr(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func isVerbose() bool {
	return getOrDefault("VERBOSE_LOG", "false") == "true"
}

func getOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}
