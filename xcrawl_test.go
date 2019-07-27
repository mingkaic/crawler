//// file: crawl_test.go

package xcrawl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/mingkaic/gardener"
	"github.com/mingkaic/stew"
)

// =============================================
//                    Globals
// =============================================

const (
	sampleYml = `
depth: 100
same_host: true
contains_tags:
- img
`
	N_TESTS = 100
)

var expectedCrawl Crawler

var sampleSite *gardener.SiteNode

// =============================================
//                    Tests
// =============================================

func TestMain(m *testing.M) {
	gard := gardener.New()
	retCode := 0
	for i := 0; i < N_TESTS && retCode == 0; i++ { // repeat all tests because of randomness
		setupExpectation(gard)
		retCode = m.Run()
	}
	os.Exit(retCode)
}

// TestNew ...
// Ensures yaml search options are parsed correctly
func TestNew(t *testing.T) {
	crawler := NewYaml([]byte(sampleYml))

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

// TestCrawlSameHost ...
// Ensures crawl visits every site in expected order
// Visit only if page has same host as root
func TestCrawlSameHost(t *testing.T) {
	crawler := NewYaml([]byte(sampleYml))
	crawler.ContainsTags = []string{}
	visited := make(map[string]struct{})
	crawler.request = func(link string) (dom *stew.Stew, err error) {
		visited[link] = struct{}{}
		// generate mock dom
		page, ok := sampleSite.Info.Pages[link]
		// ensure link in expected links
		if !ok {
			err = fmt.Errorf("unexpected link %s", link)
			return
		}
		html := gardener.ToHTML(page.Page)
		var rc io.ReadCloser = &gardener.MockRC{bytes.NewBufferString(html)}
		dom = stew.NewFromReader(rc)

		return
	}
	crawler.Crawl(sampleSite.FullLink, make(map[string]struct{}))
	baseHost := sampleSite.Hostname
	var sameHost func(*gardener.PageNode)
	sameHost = func(page *gardener.PageNode) {
		if _, ok := visited[page.FullLink]; !ok {
			t.Errorf("failed to visit link %s, %s", page.Hostname, page.LinkPath)
		}
		for _, ref := range page.Refs {
			sNode := ref.(*gardener.SiteNode)
			if sNode.Hostname == baseHost {
				sameHost(sNode.PageNode)
			}
		}
	}
}

// TestCrawlAllHosts ...
// Ensures crawl visits every site in expected order
// Visit regardless of page's hostname
func TestCrawlAllHosts(t *testing.T) {
	crawler := NewYaml([]byte(sampleYml))
	crawler.ContainsTags = []string{}
	crawler.SameHost = false
	visited := make(map[string]struct{})
	crawler.request = func(link string) (dom *stew.Stew, err error) {
		visited[link] = struct{}{}
		// generate mock dom
		page, ok := sampleSite.Info.Pages[link]
		// ensure link in expected links
		if !ok {
			err = fmt.Errorf("unexpected link %s", link)
			return
		}
		html := gardener.ToHTML(page.Page)
		var rc io.ReadCloser = &gardener.MockRC{bytes.NewBufferString(html)}
		dom = stew.NewFromReader(rc)

		return
	}
	crawler.Crawl(sampleSite.FullLink, make(map[string]struct{}))
	for link, page := range sampleSite.Info.Pages {
		if _, ok := visited[link]; !ok {
			t.Errorf("failed to visit host: %s, linkpath: %s", page.Hostname, page.LinkPath)
		}
	}
}

// =============================================
//                    Private
// =============================================

func setupExpectation(gard *gardener.Gardener) {
	expectedCrawl = Crawler{
		MaxDepth:     uint(100),
		SameHost:     true,
		ContainsTags: []string{"img"},
	}

	sampleSite = gard.GenerateSite(50)
}
