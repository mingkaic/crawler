//// file: scrape.go

// Package scrape ...
// Is a lightweight extensible web scraping package
package scrape

import (
	"golang.org/x/net/html"
	"gopkg.in/eapache/queue.v1"
)

//// public:

// ElemLookup ...
// Is a functor type for DOM-tree BFS
type ElemLookup func(*html.Node) []*html.Node

// FindAll ...
// Returns functor looking for elements with input tag
func FindAll(tag string) ElemLookup {
	return generateLookup(
		func(node *html.Node) (is_target bool, term bool) {
			term = false
			is_target = node.Data == tag
			return
		})
}

// Find ...
// Returns functor looking for a single element with input id
func Find(id string) ElemLookup {
	return generateLookup(
		func(node *html.Node) (is_target bool, term bool) {
			for _, attr := range node.Attr {
				if attr.Key == "id" {
					is_target = attr.Val == id
					break
				}
			}
			term = is_target
			return
		})
}

//// private:

// functor determines whether input node is a target
// and whether it terminates the DOM search
type queryOpt func(*html.Node) (isTarget bool, term bool)

// generates a breadth first DOM search given a query functor
func generateLookup(query queryOpt) ElemLookup {
	return func(root *html.Node) []*html.Node {
		results := []*html.Node{}
		queue := queue.New()
		queue.Add(root)

		for queue.Length() > 0 {
			curr := queue.Peek().(*html.Node)
			queue.Remove()
			isTarget, term := query(curr)
			if isTarget {
				results = append(results, curr)
			}
			if term {
				break
			}

			for child := curr.FirstChild; child != nil; child = child.NextSibling {
				queue.Add(child)
			}
		}

		return results
	}
}
