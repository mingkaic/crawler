//// file: main.go
package main

import (
	"flag"
	"io/ioutil"
	"log"

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
	options, err := ioutil.ReadFile(*cyml)
	if err != nil {
		log.Fatalf("yaml read error: %v", err)
	}

	// crawl
	crawler := crawl.New(options)
	crawler.Crawl(args[0])
}
