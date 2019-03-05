package readability

import (
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestGetContentFromOpenGraph(t *testing.T) {
	url := "https://roadsandkingdoms.com/2019/rk-insider-going-dublin/"

	doc, err := goquery.NewDocument(url)
	assert.Nil(t, err)

	c, err := getContentFromOpenGraph(doc, url)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "R&K Insider: Going to Dublin", c.Title)
	assert.Equal(t, "This week on R&K: What to know before you go to Dublin, a ridiculously calorific breakfast in Norway, and how to hunt for food in Tokyo.", c.Description)
	assert.Equal(t, "https://i1.wp.com/roadsandkingdoms.com/uploads/2019/02/Invest-in-a-good-jacket.jpg?w=2400&quality=95&strip=color&ssl=1", c.ImageURL)
}

func TestGetContentFromOpenGraphForPageWithoutOGTags(t *testing.T) {
	url := "https://tict.com.au/awards/tasmanian-tourism-awards-hall-fame/"

	doc, err := goquery.NewDocument(url)
	assert.Nil(t, err)

	c, err := getContentFromOpenGraph(doc, url)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, "", c.Title)
	assert.Equal(t, "", c.Description)
	assert.Equal(t, "", c.ImageURL)
}
