//// file: crawl_test.go

package crawl

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
search:
  depth: 10
  same_host: true
  contains_tags:
    - img
record:
  tags:
    - text
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
	if !reflect.DeepEqual(expectedCrawl.Search, crawler.Search) {
		expect, err := json.Marshal(expectedCrawl.Search)
		if err != nil {
			panic(err)
		}
		got, err := json.Marshal(crawler.Search)
		if err != nil {
			panic(err)
		}
		t.Errorf("expecting %s, got %s",
			string(expect), string(got))
	}
	if !reflect.DeepEqual(expectedCrawl.Record, crawler.Record) {
		expect, err := json.Marshal(expectedCrawl.Record)
		if err != nil {
			panic(err)
		}
		got, err := json.Marshal(crawler.Record)
		if err != nil {
			panic(err)
		}
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
		Search: struct {
			MaxDepth     uint     `yaml:"depth"`
			SameHost     bool     `yaml:"same_host"`
			ContainsTags []string `yaml:"contains_tags"`
		}{
			MaxDepth:     uint(10),
			SameHost:     true,
			ContainsTags: []string{"img"},
		},
		Record: struct {
			Tags []string
			Attr string
		}{
			Tags: []string{"text"},
			Attr: "",
		},
	}

	sampleSite = gardener.GenerateSite(50)
}
