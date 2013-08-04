package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"sync"
)

const (
	QUEUE_SIZE = 10
)

type Options struct {
	url     string
	target  string
	workers int
}

type Link struct {
	filename string
	url      string
}

func main() {
	var wg sync.WaitGroup
	opts := extractParams()

	fmt.Printf("Downloading %s\n", opts.url)
	bytes, err := fetchUrl(opts.url)
	if err != nil {
		panic(err)
	}

	links := extractLinks(bytes, opts)
	fmt.Printf("%d files\n", len(links))

	queue := make(chan Link, QUEUE_SIZE)
	for i := 0; i < opts.workers; i++ {
		wg.Add(1)
		go worker(i+1, queue, opts, &wg)
	}

	for _, link := range links {
		queue <- link
	}

	fmt.Println("Closing...")
	close(queue)
	wg.Wait()
}

func worker(index int, queue <-chan Link, opts Options, wg *sync.WaitGroup) {
	defer wg.Done()
	for link := range queue {
		fmt.Printf("Worker %d, downloading %s\n", index, link.url)
		bytes, err := fetchUrl(link.url)

		if err != nil {
			fmt.Println(err)
			continue
		}

		ioutil.WriteFile(opts.target+link.filename, bytes, 0644)
	}
}

func fetchUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.New("Failed to fetch " + url)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("Failed to parse " + url)
	}
	return body, nil
}

func extractLinks(html []byte, opts Options) []Link {
	fmt.Printf("Detecting subpaths...\n")
	r := regexp.MustCompile("(?i)<td align=top><a href=\"([^\"]+)\">")
	paths := r.FindAllSubmatch(html, -1)

	var links []Link
	for _, i := range paths {
		path := string(i[1])
		links = append(links, Link{filename: path, url: opts.url + path})
	}

	return links
}

func extractParams() Options {
	url := flag.String("u", "", "URL")
	target := flag.String("d", "/tmp", "Target directory")
	workers := flag.Int("w", 2, "Number of workers")
	flag.Parse()

	return Options{url: *url, target: *target, workers: *workers}
}
