package main

import "encoding/json"
import "fmt"
import "io"
import "net/http"
import "os"
import "strings"
import "time"

import "github.com/DavidGamba/go-getoptions"
import "github.com/hako/durafmt"
import "github.com/olorin/nagiosplugin"

func main() {
	// RIPE Atlas Base URL
	const baseurl string = "https://atlas.ripe.net/api/v2/probes/"

	// Parse options
	var probeid string
	var warnonly bool
	opt := getoptions.New()
	opt.StringVar(&probeid, "probe", "", opt.Alias("p"), opt.Required())
	opt.BoolVar(&warnonly, "warn", false)
	parseerr, _ := opt.Parse(os.Args[1:])
	if parseerr != nil {
		fmt.Printf("Failed to parse arguments: %s\n", parseerr)
		os.Exit(1)
	}

	if len(probeid) == 0 {
		fmt.Print("ERROR: No probe ID provided, aborting.\n\n")
		fmt.Print(opt.Help())
		os.Exit(1)
	}

        // Initialize Nagios module
        check := nagiosplugin.NewCheck()
        defer check.Finish()

	// Get probe status from API
	client := http.Client{ Timeout: 5 * time.Second }
	apiresp, apierr := client.Get(baseurl + probeid)
	if apierr != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Could not query API: %s", apierr))
		check.Finish()
	}
	defer apiresp.Body.Close()

	// Check the status code of the response
	if apiresp.StatusCode >= 400 {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Could not query API: %d %s", apiresp.StatusCode, http.StatusText(apiresp.StatusCode)))
		check.Finish()
	}

	// Parse the answer from the API
	var ps ProbeStatus
	apibody, _ := io.ReadAll(apiresp.Body)
	jsonerr := json.Unmarshal(apibody, &ps)
	if jsonerr != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Could not parse API response: %s", jsonerr))
		check.Finish()
	}

	// Put the time duration in a nicer format
	duration := durafmt.Parse(time.Since(ps.Status.Since)).LimitFirstN(2).String()
	if strings.ToLower(ps.Status.Name) == "connected" {
		check.AddResult(nagiosplugin.OK, fmt.Sprintf("Probe is connected for %s", duration))
		check.Finish()
	} else {
		if warnonly {
			check.AddResult(nagiosplugin.WARNING, fmt.Sprintf("Probe is in status `%s` for %s", ps.Status.Name, duration))
			check.Finish()
		} else {
			check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("Probe is in status `%s` for %s", ps.Status.Name, duration))
			check.Finish()
		}
	}
}
