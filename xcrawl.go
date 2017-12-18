//// file: crawl.go

// Package crawl ...
// Is a web crawler
package xcrawl

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/PuerkitoBio/purell"
	"github.com/mingkaic/stew"
	"gopkg.in/fatih/set.v0"
	"gopkg.in/yaml.v2"
)

//// ====== Structures ======

type ReqFunc func(string) (dom *stew.Stew, err error)

// Crawler ...
// Is the filter and record parameters
type Crawler struct {
	MaxDepth     uint     `yaml:"depth"`
	SameHost     bool     `yaml:"same_host"`
	ContainsTags []string `yaml:"contains_tags"`
	// injectables
	request ReqFunc
	record  func(*PageInfo)
}

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

type atomicInt int32

//// ====== Public ======

//// Creator & Members for Crawler

func New(ymlParams []byte) *Crawler {
	crawler := new(Crawler)
	if err := yaml.Unmarshal(ymlParams, &crawler); err != nil {
		panic(err)
	}
	return crawler
}

func (this *Crawler) Crawl(URI string) {
	// resolve uninjected functors
	if this.request == nil {
		this.request = request
	}

	// synchronization components
	queue := make(chan depthInfo)
	stopCh := make(chan struct{})
	goCount := atomicInt(0)

	// optimization components
	visited := set.New()
	visited.Add(URI)
	go func() {
		queue <- depthInfo{URI, 0}
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
		if site.depth <= this.MaxDepth {
			// propagate to linked sites
			goCount.increment() // increment in main in case goroutine completes before main
			fmt.Println("fetching", site.link, "@ depth", site.depth)
			page := this.uriProcess(site.link,
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

			if this.record != nil && page != nil {
				this.record(page)
			}
		}
	}
}

func (this *Crawler) Record(record func(*PageInfo)) {
	this.record = record
}

//// ====== Private ======

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

//// ====== Default Injectables ======

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
