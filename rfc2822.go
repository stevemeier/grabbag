package main

import "fmt"
import "time"

func main() {
	fmt.Println(rfc2822_date())
}

func rfc2822_date () (string) {
	layout := "Mon, 02 Jan 2006 15:04:05 UTC"
	time := time.Now().UTC()
	return time.Format(layout)
}
