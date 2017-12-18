//// file: crawl_test.go

package xcrawl

import (
	"os"
	"reflect"
	"testing"

	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mingkaic/gardener"
	"github.com/mingkaic/stew"
)

//// ====== Globals ======

const sampleYml = `
depth: 10
same_host: true
contains_tags:
- img
`

var expectedCrawl Crawler

var sampleSite *gardener.SiteNode

//// ====== Tests ======

func TestMain(m *testing.M) {
	setupExpectation()
	retCode := m.Run()
	os.Exit(retCode)
}

// TestNew ...
// Ensures yaml search options are parsed correctly
func TestNew(t *testing.T) {
	crawler := New([]byte(sampleYml))

	// inspect Crawler options
	expect, err := json.Marshal(expectedCrawl)
	if err != nil {
		panic(err)
	}
	got, err := json.Marshal(crawler)
	if err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(expect, got) {
		t.Errorf("expecting %s, got %s",
			string(expect), string(got))
	}
}

// TestCrawl ...
// Ensures crawl visits every site in expected order
func TestCrawl(t *testing.T) {
	crawler := New([]byte(sampleYml))
	crawler.request = func(link string) (dom *stew.Stew, err error) {
		// generate mock dom
		page, ok := sampleSite.Info.Pages[link]
		// ensure link in expected links
		if !ok {
			err = fmt.Errorf("unexpected link %s", link)
			return
		}
		html := gardener.ToHTML(page.Page)
		var rc io.ReadCloser = &gardener.MockRC{bytes.NewBufferString(html)}
		dom = stew.New(rc)

		return
	}
	crawler.Crawl(sampleSite.Link)
}

//// ====== Setup ======

func setupExpectation() {
	expectedCrawl = Crawler{
		MaxDepth:     uint(10),
		SameHost:     true,
		ContainsTags: []string{"img"},
	}

	sampleSite = gardener.GenerateSite(50)
}
