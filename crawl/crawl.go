//// file: crawl.go

// Package crawl ...
// Is a web crawler
package crawl

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/PuerkitoBio/purell"
	"github.com/mingkaic/stew"
	"gopkg.in/fatih/set.v0"
)

//// ====== Structures ======

// CrawlOpt ...
// Is the filter and record parameters
type CrawlOpt struct {
	Search struct { // search constraints
		MaxDepth     uint     `yaml:"depth"`
		SameHost     bool     `yaml:"same_host"`
		ContainsTags []string `yaml:"contains_tags"`
	}
	Record struct { // options for recording
		Tags []string
		Attr string
	}
}

// manages the depth information
type depthInfo struct {
	link  string
	depth uint
}

type atomicInt int32

//// ====== Public ======

func Crawl(origin string, opt CrawlOpt) {
	// synchronization components
	queue := make(chan depthInfo)
	stopCh := make(chan struct{})
	goCount := atomicInt(0)

	// optimization components
	visited := set.New()
	visited.Add(origin)
	go func() {
		queue <- depthInfo{origin, 0}
	}()

	go func() { // termination goroutine
		for range stopCh {
			if goCount.decrement() == 0 { // stop condition
				close(queue)
				close(stopCh)
			}
		}
	}()
	for site := range queue {
		if site.depth <= opt.Search.MaxDepth {
			// propagate to linked sites
			goCount.increment() // increment in main in case goroutine completes before main
			fmt.Println("fetching", site.link, "@ depth", site.depth)
			uriProcess(site.link, &opt,
				func(next_site string) {
					if !visited.Has(next_site) {
						visited.Add(next_site) // tag link as visited before to avoid duplicate
						goCount.increment()    // spawning new go routine
						go func() {
							queue <- depthInfo{link: next_site, depth: site.depth + 1}
							stopCh <- struct{}{} // check termination goroutine for stop condition
						}()
					}
				})
			stopCh <- struct{}{} // check termination goroutine for stop condition
		}
	}
}

//// ====== Private ======

//// Members of atomicInt

func (c *atomicInt) increment() int32 {
	return atomic.AddInt32((*int32)(c), 1)
}

func (c *atomicInt) decrement() int32 {
	return atomic.AddInt32((*int32)(c), -1)
}

func (c *atomicInt) get() int32 {
	return atomic.LoadInt32((*int32)(c))
}

//// Utilities for Crawl

// query site identified by uri for links,
// filter and handle links, and record local assets
func uriProcess(uri string, opt *CrawlOpt, linkHandle func(string)) {
	// build Stew
	dom, err := request(uri)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	// process DOM
	go func() {
		// record assets
		elems := dom.FindAll(opt.Record.Tags...)
		for _, elem := range elems {
			attrVal := elem.Attrs[opt.Record.Attr]
			fmt.Println(attrVal)
		}
	}()
	// filter links
	links := searchLinks(dom.FindAll("a"), opt)
	// validate links
	for _, link := range links {
		validLink, err := resolveRef(uri, link, opt.Search.SameHost)
		if err == nil {
			linkHandle(validLink)
		}
	}
}

// todo: make request injectable
// construct stew dom tree from custom request to link
func request(link string) (dom *stew.Stew, err error) {
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
	dom = stew.New(resp.Body)
	return
}

// filter link given options
func searchLinks(elems []*stew.Stew, opt *CrawlOpt) []string {
	links := []string{}
	for _, elem := range elems {
		contains := false
		for _, contTag := range opt.Search.ContainsTags {
			contains = contains || elem.Descs[contTag] != nil
			if contains {
				break
			}
		}
		if len(opt.Search.ContainsTags) == 0 || contains {
			links = append(links, elem.Attrs["href"]...)
		}
	}
	return links
}

// validate and normalize links
func resolveRef(base, ref string, SameHost bool) (link string, err error) {
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
		} else if SameHost && hostname != baseURL.Hostname() {
			err = fmt.Errorf("external hostname: %s", hostname)
		} else {
			link = resURL.String()
		}
	}
	return
}
