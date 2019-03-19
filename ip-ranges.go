package main

import "encoding/json"
import "fmt"
import "io/ioutil"
import "os"
import "strings"
import "net/http"
import "time"

// from https://mholt.github.io/json-to-go/
type AutoGenerated struct {
	SyncToken  string `json:"syncToken"`
	CreateDate string `json:"createDate"`
	Prefixes   []struct {
		IPPrefix string `json:"ip_prefix"`
		Region   string `json:"region"`
		Service  string `json:"service"`
	} `json:"prefixes"`
}

func main() {

	var httpClient = &http.Client{
		Timeout: time.Second * 5,
	}

	response, err := httpClient.Get("https://ip-ranges.amazonaws.com/ip-ranges.json")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	byteValue, _ := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var data AutoGenerated
	json.Unmarshal(byteValue, &data)

	for x := range data.Prefixes {
		if strings.EqualFold(data.Prefixes[x].Service, "CLOUDFRONT") {
			fmt.Println(data.Prefixes[x].IPPrefix)
		}
	}
}
