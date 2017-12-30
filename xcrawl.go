//// file: crawl.go

// Package crawl ...
// Is a web crawler
package xcrawl

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"github.com/PuerkitoBio/purell"
	"github.com/mingkaic/phantomgo"
	"github.com/mingkaic/stew"
	"gopkg.in/fatih/set.v0"
	"gopkg.in/yaml.v2"
)

// =============================================
//                    Declarations
// =============================================

// Crawler ...
// Is the filter and record parameters
type Crawler struct {
	MaxDepth     uint     `yaml:"depth",json:"depth"`
	SameHost     bool     `yaml:"same_host",json:"sameHost"`
	ContainsTags []string `yaml:"contains_tags",json:"containsTags"`
	// injectables
	request ReqFunc
	record  RecFunc // optional for recording page information
}

type VisitCtx interface {
	Add(...interface{})
	Has(...interface{}) bool
}

// PageInfo ...
// Represents useful data for a single page
type PageInfo struct {
	DOM  *stew.Stew
	Link string
	Refs *set.Set
}

// manages the depth information
type depthInfo struct {
	link  string
	depth uint
}

type ReqFunc func(string) (*stew.Stew, error)

type RecFunc func(*sync.WaitGroup, *PageInfo)

// =============================================
//                    Public
// =============================================

//// Creator & Members for Crawler

// NewYaml ...
// Marshal yaml options as Crawler parameters
func NewYaml(ymlParams []byte) *Crawler {
	crawler := new(Crawler)
	if err := yaml.Unmarshal(ymlParams, &crawler); err != nil {
		panic(err)
	}
	return crawler
}

// NewJson ...
// Marshal json options as Crawler parameters
func NewJson(jsonParams []byte) *Crawler {
	crawler := new(Crawler)
	if err := json.Unmarshal(jsonParams, &crawler); err != nil {
		panic(err)
	}
	return crawler
}

// Crawl ...
// Visits all pages starting from input URI
func (this *Crawler) Crawl(URI string, visited VisitCtx) {
	// resolve uninjected functors
	if this.request == nil {
		this.request = StaticRequest
	}

	// synchronization components
	queue := make(chan depthInfo)
	var wg sync.WaitGroup

	// optimization components
	visited.Add(URI)
	wg.Add(1) // wait until initial uri is processed
	go func() {
		queue <- depthInfo{URI, 0}
	}() // not passing queue and URI as argument since this routine is guaranteed to finish before end of channel

	go func() { // termination goroutine
		wg.Wait()
		close(queue)
	}() // not passing anything here because this routine determines the end of channel
	for site := range queue {
		if site.depth <= this.MaxDepth {
			// propagate to linked sites
			page := this.uriProcess(site.link,
				func(next_site string) {
					if !visited.Has(next_site) {
						visited.Add(next_site) // tag link as visited before to avoid duplicate
						wg.Add(1)              // wait until next_site is processed
						go func(next_site string, depth uint) {
							queue <- depthInfo{link: next_site, depth: depth}
						}(next_site, site.depth+1) // termination is dependent on this go routine's completion
					}
				})

			if this.record != nil && page != nil {
				wg.Add(1) // wait on record
				go this.record(&wg, page)
			}
		}
		wg.Done() // site is processed
	}
}

// InjectReq ...
// Adds request functor
func (this *Crawler) InjectReq(req ReqFunc) {
	this.request = req
}

// InjectRec ...
// Adds record functor
func (this *Crawler) InjectRec(rec RecFunc) {
	this.record = rec
}

//// Injectables

// StaticRequest ...
// Construct stew dom tree from page before javascript execution
// Default Request Injectable
func StaticRequest(link string) (dom *stew.Stew, err error) {
	// disable ssl verification
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	dom = stew.NewFromRes(resp)
	return
}

// GetDynamicRequest ...
// Returns DynamicRequest, which Construct stew dom from page after javascript execution
func GetDynamicRequest(execPath string) ReqFunc {
	browser := phantomgo.NewPhantom(execPath, "Mozilla/5.0")
	return func(link string) (dom *stew.Stew, err error) {
		p := &phantomgo.Param{
			Method:       "GET", //POST or GET ..
			Url:          link,
			UsePhantomJS: true,
		}
		resp, err := browser.Download(p)
		if err != nil {
			return
		}
		dom = stew.NewFromRes(resp)
		return
	}
}

// =============================================
//                    Private
// =============================================

//// Private Members for Crawler

// query site identified by uri for links,
// filter and handle links, and record local assets
func (this *Crawler) uriProcess(uri string, handleLink func(string)) *PageInfo {
	// build Stew
	dom, err := this.request(uri)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	// filter links
	links := this.searchLinks(dom.FindAll("a"))
	refs := set.New()
	// validate links
	for _, link := range links {
		validLink, err := this.resolveRef(uri, link)
		if err == nil {
			handleLink(validLink.String())
		}
		if validLink != nil {
			refs.Add(validLink.String())
		}
	}
	return &PageInfo{dom, uri, refs}
}

// filter link given options
func (this *Crawler) searchLinks(elems []*stew.Stew) []string {
	links := []string{}
	for _, elem := range elems {
		contains := false
		for _, contTag := range this.ContainsTags {
			contains = contains || elem.Descs[contTag] != nil
			if contains {
				break
			}
		}
		if len(this.ContainsTags) == 0 || contains {
			links = append(links, elem.Attrs["href"]...)
		}
	}
	return links
}

// validate and normalize links
func (this *Crawler) resolveRef(base, ref string) (link *url.URL, err error) {
	normalFlag := purell.FlagsUnsafeGreedy
	refURL, err := url.Parse(purell.MustNormalizeURLString(ref, normalFlag))
	if err != nil {
		return
	}
	baseURL, err := url.Parse(purell.MustNormalizeURLString(base, normalFlag))
	if err == nil {
		resURL := baseURL.ResolveReference(refURL)
		hostname := resURL.Hostname()
		if len(hostname) == 0 {
			err = fmt.Errorf("invalid uri: %s", link)
		} else {
			if this.SameHost && hostname != baseURL.Hostname() {
				err = fmt.Errorf("external hostname: %s", hostname)
			}
			link = resURL
		}
	}
	return
}
