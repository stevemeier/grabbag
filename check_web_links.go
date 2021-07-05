package main

import (
	"fmt"
	"golang.org/x/net/html"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Source:
// https://schier.co/blog/a-simple-web-scraper-in-go

// Helper function to pull the href attribute from a Token
func getHref(t html.Token) (ok bool, href string) {
	// Iterate over token attributes until we find an "href"
	for _, a := range t.Attr {
		if a.Key == "href" {
			href = a.Val
			ok = true
		}
	}

	// "bare" return will return the variables (ok, href) as 
	// defined in the function definition
	return
}

// Extract all http** links from a given webpage
func crawl(url string, ch chan string, chFinished chan bool) {
	resp, err := http.Get(url)

	defer func() {
		// Notify that we're done after this function
		chFinished <- true
	}()

	if err != nil {
//		fmt.Println("ERROR: Failed to crawl:", url)
		return
	}

	b := resp.Body
	defer b.Close() // close Body when the function completes

	z := html.NewTokenizer(b)

	for {
		tt := z.Next()

		switch {
		case tt == html.ErrorToken:
			// End of the document, we're done
			return
		case tt == html.StartTagToken:
			t := z.Token()

			// Check if the token is an <a> tag
			isAnchor := t.Data == "a"
			if !isAnchor {
				continue
			}

			// Extract the href value, if there is one
			ok, url := getHref(t)
			if !ok {
				continue
			}

			// Make sure the url begines in http**
//			hasProto := strings.Index(url, "http") == 0
//			isRelative := strings.Index(url, "/") == 0
//			if hasProto || isRelative {
				ch <- url
//			}
		}
	}
}

func main() {
        defer func() {
                if err := recover(); err != nil {
			fmt.Printf("UNKNOWN: %s\n", err)
                        os.Exit(3)
                }
        }()

	foundUrls := make(map[string]bool)
	startUrl := os.Args[1]

	// Check Start URL
	su, err := url.Parse(startUrl)
	if err != nil {
		fmt.Printf("UNKNOWN: %s\n", err)
		os.Exit(3)
	}

	// Channels
	chUrls := make(chan string)
	chFinished := make(chan bool)

	// Kick off the crawl process (concurrently)
	go crawl(startUrl, chUrls, chFinished)

	// Subscribe to both channels
	for c := 0; c < 1; {
		select {
		case url := <-chUrls:
			foundUrls[url] = true
		case <-chFinished:
			c++
		}
	}

	// We're done! Print the results...
	var broken int
	for url, _ := range foundUrls {
//		fmt.Printf("Completing %s\n", url)
//		url = completeURL(startUrl, url)
		url = completeURL(su, url)
		if strings.Index(url, "http") != 0 {
			continue
		}

//		fmt.Printf("Checking %s\n", url)
		if !linkOK(url) {
			broken++
			fmt.Printf("%s is broken\n", url)
		}
	}

	close(chUrls)

	if broken > 0 {
		fmt.Printf("CRITICAL: %d broken links found\n", broken)
		os.Exit(2)
	} else {
		fmt.Printf("OK: No broken links found (%d ok)\n", len(foundUrls))
		os.Exit(0)
	}
}

func linkOK (url string) (bool) {
	resp, err := http.Head(url)
	if err != nil { return false }
	if resp.StatusCode  < 400 { return true }
	if resp.StatusCode == 405 { return true }

//	fmt.Println(resp)

	// Cloudflare
	if resp.Header.Get("server") == "cloudflare" && resp.StatusCode == 403 {
		return true
	}

	// Akamai
	if resp.Header.Get("server") == "AkamaiGHost" && resp.StatusCode == 503 {
		return true
	}

	return false
}

//func completeURL (base string, href string) (string) {
//	b, _ := url.Parse(base)
func completeURL (b *url.URL, href string) (string) {
	u, err := url.Parse(href)
	if err != nil { return "" }

	// return absolute URLs unchanged
	if u.IsAbs() { return href }

	// Scheme is missing
	if u.Scheme == "" {
		u.Scheme = b.Scheme
	}

	// Host is missing
	if u.Host == "" {
		u.Host = b.Host
	}

	return fmt.Sprintf("%s", u)
}
