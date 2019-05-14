package main

// Author: Steve Meier
// Date: 2019-05-14

import "github.com/DavidGamba/go-getoptions"
import "fmt"
import "io/ioutil"
import "net/http"
import "os"

const Version string = "20190514"

func main () {
	var url string
	var warning int
	var critical int

        opt := getoptions.New()
        opt.StringVar(&url, "url", "", opt.Description("URL to retrieve"), opt.ArgName("URL"))
        opt.IntVar(&warning, "w", 0, opt.Description("Warning if less than"), opt.ArgName("bytes"))
        opt.IntVar(&critical, "c", 0, opt.Description("Critical if less than"), opt.ArgName("bytes"))
	_, err := opt.Parse(os.Args[1:])

	// Parsing parameters failed
	if err != nil {
		fmt.Printf("UNKNOWN: Failed to parse options: %v\n", err)
		os.Exit(3)
	}

	// Without parameters, print help
	if len(os.Args[1:]) == 0 {
		fmt.Print(opt.Help())
		os.Exit(0)
	}

	// No URL, no fun
	if url == "" {
		fmt.Println("UNKNOWN: No URL defined")
		os.Exit(3)
	}

	size, err := get_size(url)

	if size == -1 {
		fmt.Printf("UNKNOWN: Failed to fetch URL: %v\n", err)
		os.Exit(3)
	}

	if size < critical {
		fmt.Printf("CRITICAL: Size is only %d bytes\n", size)
		os.Exit(2)
	}

	if size < warning {
		fmt.Printf("WARNING: Size is only %d bytes\n", size)
		os.Exit(1)
	}

	fmt.Printf("OK: Size is %d bytes\n", size)
	os.Exit(0)
}

func get_size (url string) (int, error) {
        client := &http.Client{}
        request, _ := http.NewRequest("GET", url, nil)
        request.Header.Set("User-Agent", "check_errata_size/" + Version)
        resp, err := client.Do(request)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return -1, err
	}

	return len(data), nil
}
