//// file: main.go
package main

import (
	"flag"
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
	"github.com/mingkaic/xcrawl/crawl"
	//"github.com/mingkaic/xcrawl/model"
)

func main() {
	// argument sanitation
	flag.Parse()
	args := flag.Args()
	cyml := flag.String("cyml", "media_crawl.yml",
		"yml file outlining search constraint")
	if len(args) < 1 {
		log.Fatalf("Specify starting location")
	}

	// db prepare
	//model.InitDB("sqlite3", "test.db")
	//defer model.Db.Close()

	// search parameter components
	opt := crawl.CrawlOpt{}
	options, err := ioutil.ReadFile(*cyml)
	if err != nil {
		log.Fatalf("yaml read error: %v", err)
	}
	if err := yaml.Unmarshal(options, &opt); err != nil {
		log.Fatalf("unmarshal error: %v", err)
	}

	// crawl
	crawl.Crawl(args[0], opt)
}
