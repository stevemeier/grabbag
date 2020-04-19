package main

import "compress/bzip2"
import "encoding/json"
import "encoding/xml"
import "fmt"
import "io/ioutil"
import "github.com/DavidGamba/go-getoptions"
import "github.com/hashicorp/logutils"
import "github.com/kolo/xmlrpc"
import "log"
import "os"
import "regexp"
import "strings"
import "strconv"
import "syscall"
import "time"
import "net"

// These two need to be loaded if cert-check is to be disabled
import "net/http"
import "crypto/tls"

// URLs (used in --simple mode)
const errataurl = "https://cefs.steve-meier.de/errata.latest.json.bz2"
const rhsaovalurl = "https://www.redhat.com/security/data/oval/com.redhat.rhsa-all.xml.bz2"

const Version int = 20200419
const timelayout = "2006-01-02 15:04:05"
var SupportedAPI = []float64{10.9,  // Spacewalk 0.6
                             10.10, // Spacewalk 0.7
                             10.11, // Spacewalk 0.8 through 1.1
			     10.15, // Spacewalk 1.2
			     10.16, // Spacewalk 1.3 and 1.4
			     11.00, // Spacewalk 1.5
			     11.1,  // Spacewalk 1.6 through 1.8
			     12,    // Spacewalk 1.9
			     13,    // Spacewalk 2.0
			     13.0,
			     14,    // Spacewalk 2.1
			     14.0,
			     15,    // Spacewalk 2.2
			     15.0,
			     16,    // Spacewalk 2.3
			     16.0,
			     17,    // Spacewalk 2.4
			     17.0,
			     18,    // Spacewalk 2.5
			     18.0,
			     19,    // Spacewalk 2.6
			     19.0,
			     20,    // Spacewalk 2.7
			     20.0,
			     21,    // Spacewalk 2.8
			     21.0,
			     22,    // Spacewalk 2.9
			     22.0,
			     23,    // Spacewalk 2.10
			     23.0,
		    }

type Meta struct {
	Author		string
	Disclaimer	string
	License		string
	Timestamp	string
}

type Erratum struct {
	Id		string		`json:"id"`		// Only needed in array approach
	Description	string		`json:"description"`
	From		string		`json:"from"`
	IssueDate	string		`json:"issue_date"`
	Keywords	[]string	`json:"keywords"`
	Manual		string		`json:"manual"`
	Notes		string		`json:"notes"`
	OsArch		[]string	`json:"os_arch"`
	OsRelease	[]string	`json:"os_release"`
	Packages	[]string	`json:"packages"`
	Product		string		`json:"product"`
	References	string		`json:"references"`
	Release		string		`json:"release"`
	Severity	string		`json:"severity"`
	Solution	string		`json:"solution"`
	Synopsis	string		`json:"synopsis"`
	Topic		string		`json:"topic"`
	Type		string		`json:"type"`
}

type Raw struct {
	Advisories	[]Erratum
	Meta		Meta
}

type SWerrata struct {
	Synopsis	string	`xmlrpc:"synopsis"`
	AdvisoryName	string	`xmlrpc:"advisory_name"`
	AdvisoryRelease	int	`xmlrpc:"advisory_release"`
	AdvisoryType	string	`xmlrpc:"advisory_type"`
	From		string	`xmlrpc:"errataFrom"`
	Product		string	`xmlrpc:"product"`
	Topic		string	`xmlrpc:"topic"`
	Description	string	`xmlrpc:"description"`
	References	string	`xmlrpc:"references"`
	Notes		string	`xmlrpc:"notes"`
	Solution	string	`xmlrpc:"solution"`
}

// The Url field is not supported in all versions of Spacewalk
// Version 1.3 and newer seems to support it
type Bugzilla struct {
	Text string	`xml:",chardata" xmlrpc:"summary"`
	Href string	`xml:"href,attr" xmlrpc:"url"`
	ID   int64	`xml:"id,attr" xmlrpc:"id"`
}

type Inventory struct {
	filename2id	map[string]int64
	id2channels	map[int64][]string
	id2filename	map[int64]string
}

type OvalData struct {
	Description	string
	References	[]string
	Rights		string
	Bugs		[]Bugzilla
}

func main () {
	var debug bool
	var quiet bool
	var autopush bool
	var publish bool
	var server string

	var created int
	var updated int

	var security bool
	var bugfix bool
	var enhancement bool

	var ignoreapiversion bool
	var protocol string
	var insecure bool

	var inchannels *[]string
	var exchannels *[]string

	var syncchannels bool
	var synctimeout int

	var exerrata *[]string

	var erratafile string
	var rhsaovalfile string

	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	opt.BoolVar(&quiet, "quiet", false)
	opt.StringVar(&server, "server", "localhost")
	opt.BoolVar(&publish, "publish", false)
	opt.BoolVar(&autopush, "autopush", false)

	opt.BoolVar(&security, "security", false)
	opt.BoolVar(&bugfix, "bugfix", false)
	opt.BoolVar(&enhancement, "enhancement", false)

	opt.BoolVar(&ignoreapiversion, "ignore-api-version", false)
	opt.StringVar(&protocol, "protocol", "http")
	opt.BoolVar(&insecure, "insecure", false)

	inchannels = opt.StringSlice("include-channels", 1, 255)
	exchannels = opt.StringSlice("exclude-channels", 1, 255)

	opt.BoolVar(&syncchannels, "sync-channels", false)
	opt.IntVar(&synctimeout, "sync-timeout", 600)

	exerrata = opt.StringSlice("exclude-errata", 1, 255)

	opt.StringVar(&erratafile, "errata", "")
	opt.StringVar(&rhsaovalfile, "rhsa-oval", "")

	// Parse options
	remaining, err := opt.Parse(os.Args[1:])

	// Set up logger
	filter := &logutils.LevelFilter{
		Levels: []logutils.LogLevel{"DEBUG","INFO","WARNING","ERROR"},
		MinLevel: logutils.LogLevel(min_log_level(debug, quiet)),
		Writer: os.Stdout,
	}

	// Set up log filter
	log.SetOutput(filter)

	log.Printf("[DEBUG] Version is %d\n", Version)
	if len(os.Args[1:]) == 0 {
		log.Print(opt.Help())
		os.Exit(4)
	}
	if err != nil {
		log.Printf("[ERROR] Failed to parse options: %v\n", err)
		os.Exit(4)
	}
	if len(remaining) > 0 {
		log.Printf("[ERROR] The following options are unrecognized: %v\n", remaining)
		os.Exit(4)
	}

	// Check if running as root
	if running_as_root() {
		log.Println("[INFO] Running as root is not recommended!")
	}

	// --autopush is deprecated
	if autopush {
		log.Println("[INFO] The --autopush option is deprecated and has no effect anymore")
	}

	// If no errata type is selected, enable all
	if (!(security || bugfix || enhancement)) {
		security, bugfix, enhancement = true, true, true
	}

	// Load errata data
	var allerrata Raw
	if erratafile != "" {
		log.Printf("[INFO] Loading errata data from %s\n", erratafile)
		allerrata = ParseErrata(*file_content(erratafile))
	} else {
		log.Printf("[INFO] Loading errata data from %s\n", errataurl)
		allerrata = ParseErrata(*download_bzip2(errataurl))
	}
	if len(allerrata.Advisories) > 0 {
		log.Printf("[INFO] Loaded %d advisories from errata file\n", len(allerrata.Advisories))
	} else {
		log.Printf("[ERROR] Could not parse errata data from %s\n", erratafile)
		os.Exit(5)
	}

	// Load Red Hat OVAL data
	var oval map[string]OvalData
	if rhsaovalfile != "" {
		log.Printf("[INFO] Loading RHSA oval data from %s\n", rhsaovalfile)
		oval = ParseOval(*file_content(rhsaovalfile))
	} else {
		log.Printf("[INFO] Loading RHSA oval data from %s\n", rhsaovalurl)
		oval = ParseOval(*download_bzip2(rhsaovalurl))
	}
	if len(oval) > 0 {
		log.Printf("[INFO] Loaded %d datasets from Red Hat OVAL file\n", len(oval))
	}

	// Configure timeout
	// Source: https://medium.com/@nate510/don-t-use-go-s-default-http-client-4804cb19f779
	// and TLS options
	// Source: https://stackoverflow.com/questions/12122159/how-to-do-a-https-request-with-bad-certificate
	var netTransport = &http.Transport{ Dial: (&net.Dialer{ Timeout: 5 * time.Second, }).Dial,
	TLSHandshakeTimeout: 5 * time.Second,
	TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, }

	// Create XML-RPC client
	client, err := xmlrpc.NewClient(protocol + "://" + server + "/rpc/api", netTransport)
	if err != nil {
		log.Println("[ERROR] Could not create XML-RPC client: ", err.Error())
		os.Exit(2)
	}

	// Get server version
	var apiversion string
	err = client.Call("api.get_version", nil, &apiversion)
	if err != nil {
		if strings.Contains(err.Error(), "cannot validate certificate") {
			log.Println("[ERROR] Certicate verification failed. Use --insecure if you have a self-signed cert")
			os.Exit(6)
		}
		if strings.Contains(err.Error(), "i/o timeout") {
			log.Println("[ERROR] Timeout connecting to server")
			os.Exit(6)
		}
		log.Printf("[ERROR] Could not determine server version: %v\n", err)
		os.Exit(2)
	}

	if (!check_api_support(apiversion, SupportedAPI) && !ignoreapiversion) {
		log.Printf("[ERROR] API version %s is not supported!\n", apiversion)
		os.Exit(3)
	} else {
		log.Printf("[INFO] API version %s is supported", apiversion)
	}

	// Read and check credentials
	username := os.Getenv("SPACEWALK_USER")
	password := os.Getenv("SPACEWALK_PASS")

	if (username == "") || (password == "") {
		log.Println("[ERROR] Credentials not set!")
		os.Exit(3)
	}

	// Authenticate and get sessionKey
	// Setup defered closing of session
	var sessionkey string = init_session(client, username, password)
	if sessionkey == "" {
		log.Println("[ERROR] Authentication failed!")
		os.Exit(1)
	}
	defer close_session(client, sessionkey)

	// Check admin status
	if publish {
		if (user_is_admin(client, sessionkey, username)) {
			log.Printf("[INFO] User %s has administrator access to this server\n", username)
		} else {
			log.Printf("[ERROR] User %s does NOT have administrator access", username);
			log.Println("[ERROR] You have set --publish but your user has insufficient access rights");
			log.Println("[ERROR] Either use an account that is Satellite/Org/Channel Administator privileges or omit --publish");
			os.Exit(1)
		}
	}

	// List all channels
	var channels []string = get_channel_list(client, sessionkey)

	// Handle channel includes and excludes
	channels = include_channels(channels, inchannels)
	channels = exclude_channels(channels, exchannels)

	// Check that there are still channels left
	if len(channels) == 0 {
		log.Println("[ERROR] All channels have been excluded")
		os.Exit(8)
	}

	// Sync channels, if requested
	if string_to_float(apiversion) < 11 {
		log.Println("[INFO] This API version does not support synching")
		syncchannels = false
	}
	if syncchannels {
		for _, channel := range channels {
			log.Printf("[INFO] Starting Repository Sync for channel %s\n", channel)
			if channel_sync_repo(client, sessionkey, channel) {
				syncstart := time.Now()
				for {
					time.Sleep(30 * time.Second)
					if get_channel_last_sync(client, sessionkey, channel).After(syncstart) {
						// Sync finished
						log.Printf("[INFO] Sync for channel %s finished\n", channel)
						break
					}
					if time.Now().After(syncstart.Add(time.Duration(synctimeout) * time.Second)) {
						// Sync failed
						log.Printf("[ERROR] Repository Sync for channel %s failed!\n", channel)
						break
					}
				}
			}
		}
	}

	// Get packages of channel
	log.Println("[INFO] Getting server inventory")
	var inv Inventory = get_inventory(client, sessionkey, channels)

	// Get existing errata
	var existing = get_existing_errata(client, sessionkey, channels)

	// Process errata
	for _, errata := range allerrata.Advisories {

		if errata_is_excluded(errata.Id, exerrata) {
			log.Printf("[INFO] Excluding %s\n", errata.Id)
			continue
		}

		if (errata.Type == "Security Advisory" && !security) {
			log.Printf("[INFO] Skipping %s\n", errata.Id)
			continue
		}
		if (errata.Type == "Bug Fix Advisory" && !bugfix) {
			log.Printf("[INFO] Skipping %s\n", errata.Id)
			continue
		}
		if (errata.Type == "Product Enhancement Advisory" && !enhancement) {
			log.Printf("[INFO] Skipping %s\n", errata.Id)
			continue
		}

		log.Printf("[INFO] Processing %s\n", errata.Id)

		var pkglist []int64 = get_packages_for_errata(errata, inv)

		if len(pkglist) == 0 {
			log.Printf("[INFO] Skipping errata %s (%s) -- No packages found\n", errata.Id, errata.Synopsis);
			continue
		}

		var chanlist []string = get_channels_of_packages(pkglist, inv)

		var info SWerrata
		info.AdvisoryName = errata.Id
		info.AdvisoryType = errata.Type
		info.Synopsis = errata.Synopsis
		info.Description = errata.Description
		info.Product = errata.Product
		info.References = errata.References
		info.Solution = errata.Solution
		info.Topic = errata.Topic
		info.Notes = errata.Notes
		info.From = errata.From

		// If Red Hat Oval data is available, use it
		if oval[(errata.Id)].Description != "" {
			info.Description = oval[(errata.Id)].Description
		}
		if oval[(errata.Id)].Rights != "" {
			info.Notes = oval[(errata.Id)].Rights
		}

		var success bool
		if exists := existing[(errata.Id)]; !exists {
			// Create Errata
			log.Printf("[INFO] Creating errata for %s (%s) (%d of %d)\n", errata.Id, errata.Synopsis, len(pkglist), len(errata.Packages))
			if string_to_float(apiversion) >= 10.16 {
				success = create_errata(client, sessionkey, info, oval[(errata.Id)].Bugs, errata.Keywords, pkglist, false, []string{})
				if success { created++ }
			} else {
				success = create_errata(client, sessionkey, info, []Bugzilla{}, errata.Keywords, pkglist, false, []string{})
				if success { created++ }
			}

			if string_to_float(apiversion) >= 12 {
				log.Printf("[INFO] Adding issue date to %s\n", errata.Id)
				issuedate, _ := time.Parse(timelayout, errata.IssueDate)
				success = add_issue_date(client, sessionkey, errata.Id, issuedate)
				if !success { log.Printf("[ERROR] Adding issue date to %s failed\n", errata.Id) }
			}

			if string_to_float(apiversion) >= 21 && errata.Severity != "" {
				log.Printf("[INFO] Adding severity %s to %s\n", errata.Severity, errata.Id)
				success = add_severity(client, sessionkey, errata.Id, errata.Severity)
				if !success { log.Printf("[ERROR] Adding severity to %s failed\n", errata.Id) }
			}

			if publish {
				for _, singlechannel := range chanlist {
					log.Printf("[INFO] Publishing %s to channel %s\n", errata.Id, singlechannel)
					success = publish_errata(client, sessionkey, errata.Id, []string{singlechannel})
					if !success { log.Printf("[ERROR] Publishing %s to channel %s failed\n", errata.Id, singlechannel) }
				}
				if errata.Type == "Security Advisory" && oval[(errata.Id)].References != nil {
					log.Printf("[INFO] Adding CVE information to %s\n", errata.Id)
					success = add_cve_to_errata(client, sessionkey, info, oval[(errata.Id)].References)
					if !success { log.Printf("[INFO] Adding CVE information to %s failed\n", errata.Id) }
				}
			}

		} else {
			// Update Errata
			var curlist []int64 = list_packages(client, sessionkey, errata.Id)
			var newlist []int64 = only_in_first_int64(pkglist, curlist)

			if len(pkglist) > len(curlist) {
				log.Printf("[INFO] Adding packages to %s\n", errata.Id)
				var pkgsadded int64 = add_packages(client, sessionkey, errata.Id, newlist)
				if pkgsadded > 0 { updated++ }

				if publish {
					for _, channel := range get_channels_of_packages(newlist, inv) {
						log.Printf("[INFO] Republishing %s to channel %s\n", errata.Id, channel)
						success = publish_errata(client, sessionkey, errata.Id, []string{channel})
						if !success { log.Printf("[ERROR] Republishing %s to channel %s failed\n", errata.Id, channel) }
					}
				}
			}
		}
		// Restore channel membership of package after publishing to multiple channels
		if publish && len(chanlist) > 1 {
			for _, pkg := range pkglist {
				oldchannels := inv.id2channels[pkg]
				nowchannels := list_providing_channels(client, sessionkey, pkg)
				pushedto := only_in_first_string(nowchannels, oldchannels)
				for _, channel := range pushedto {
					log.Printf("[INFO] Removing package %s from channel %s\n", inv.id2filename[pkg], channel)
					remove_packages_from_channel(client, sessionkey, channel, []int64{pkg})
				}
			}
		}
	}

	log.Printf("[INFO] Errata created: %d\n", created);
	log.Printf("[INFO] Errata updated: %d\n", updated);

	if !publish && created > 0 {
		log.Println("[INFO] Errata have been created but NOT published!");
		log.Println("[INFO] Please go to: Errata -> Manage Errata -> Unpublished to find them");
		log.Println("[INFO] If you want to publish them please delete the unpublished Errata and run this script again");
		log.Println("[INFO] with the --publish parameter");
	}

	os.Exit(0)
}

func init_session (client *xmlrpc.Client, username string, password string) string {
	params := make([]interface{}, 2)
	params[0] = username
	params[1] = password

	var sessionkey string
	err := client.Call("auth.login", params, &sessionkey)

	if err != nil {
		return ""
	}

	return sessionkey
}

func close_session (client *xmlrpc.Client, sessionkey string) bool {
	params := make([]interface{}, 1)
	params[0] = sessionkey

	err := client.Call("auth.logout", params, nil)
	return err == nil
}

func user_is_admin (client *xmlrpc.Client, sessionkey string, username string) bool {
	params := make([]interface{}, 2)
	params[0] = sessionkey
	params[1] = username

	var roles []string
	err := client.Call("user.list_roles", params, &roles)

	if err != nil {
		return false
	}

	for _, role := range roles {
		if (role == "satellite_admin" || role == "org_admin" || role == "channel_admin") {
			return true
		}
	}

	return false
}

func get_channel_list (client *xmlrpc.Client, sessionkey string) []string {
	params := make([]interface{}, 1)
	params[0] = sessionkey

	var channels []interface{}
	err := client.Call("channel.list_all_channels", params, &channels)

	var channelnames []string
	if err != nil {
		return channelnames
	}

	for _, channel := range channels {
		if details, ok := channel.(map[string]interface{}); ok {
			channelnames = append(channelnames, details["label"].(string))
		}
	}

	return channelnames
}

func get_inventory (client *xmlrpc.Client, sessionkey string, channels []string) Inventory {
	params := make([]interface{}, 2)

	var inv Inventory
	inv.filename2id = make(map[string]int64)
	inv.id2channels = make(map[int64][]string)
	inv.id2filename = make(map[int64]string)
	for _, channel := range channels {
		params[0] = sessionkey
		params[1] = channel

		var packages []interface{}
		log.Printf("[INFO] Scanning channel %s\n", channel)
		err := client.Call("channel.software.list_all_packages", params, &packages)
		if err != nil {
			return inv
		}

		for _, pkg := range packages {
			if details, ok := pkg.(map[string]interface{}); ok {
				id := details["id"].(int64)
				filename, inchannels := get_package_details(client, sessionkey, id)
				inv.filename2id[filename] = id
				inv.id2channels[id] = inchannels
				inv.id2filename[id] = filename
			}
		}

	}

	return inv
}

func get_existing_errata (client *xmlrpc.Client, sessionkey string, channels []string) map[string]bool {
	params := make([]interface{}, 2)
	params[0] = sessionkey

	existing := make(map[string]bool)

	type Response struct {
		Id			int64	`xmlrpc:"id"`
		Date			string	`xmlrpc:"date"`
		AdvisoryType		string	`xmlrpc:"advisory_type"`
		AdvisoryName		string	`xmlrpc:"advisory_name"`
		AdvisorySynopsis	string	`xmlrpc:"advisory_synopsis"`
		Advisory		string	`xmlrpc:"advisory"`
		IssueDate		string	`xmlrpc:"issue_date"`
		UpdateDate		string	`xmlrpc:"update_date"`
		Synopsis		string	`xmlrpc:"synopsis"`
		LastModified		string	`xmlrpc:"last_modified_date"`
	}
	var response []Response

	type Unpub struct {
		Id			int64	`xmlrpc:"id"`
		Published		int64	`xmlrpc:"published"`
		Advisory		string	`xmlrpc:"advisory"`
		AdvisoryName		string	`xmlrpc:"advisory_name"`
		AdvisoryType		string	`xmlrpc:"advisory_type"`
		Synopsis		string	`xmlrpc:"synopsis"`
		Created			time.Time	`xmlrpc:"created"`
		UpdateDate		time.Time	`xmlrpc:"update_date"`
	}

	var unpub []Unpub
	log.Println("[INFO] Fetching unpublished errata")
	err := client.Call("errata.list_unpublished_errata", params, &unpub)
	if err != nil {
		return existing
	}

	for _, errata := range unpub {
		existing[(errata.AdvisoryName)] = true
	}

	for _, channel := range channels {
		params[1] = channel
		log.Printf("[INFO] Fetching existing errata for channel %s\n", channel)

		err := client.Call("channel.software.list_errata", params, &response)
		if err != nil {
			return existing
		}

		for _, errata := range response {
			existing[(errata.AdvisoryName)] = true
		}
	}

	return existing
}

func get_package_details (client *xmlrpc.Client, sessionkey string, id int64) (string, []string) {
	params := make([]interface{}, 2)
	params[0] = sessionkey
	params[1] = id

	var details interface{}
	var inchannels []string
	err := client.Call("packages.get_details", params, &details)
	if err != nil {
		return "", []string{}
	}

	if detail, ok := details.(map[string]interface{}); ok {
		for _, provchan := range detail["providing_channels"].([]interface{}) {
			inchannels = append(inchannels, provchan.(string))
		}
		return detail["file"].(string), inchannels
	}

	return "", []string{}
}

func ParseErrata (data []byte) Raw {
	var allerrata Raw
	err := json.Unmarshal(data, &allerrata)
	if err != nil {
		log.Printf("[ERROR] Parsing JSON data failed: %v\n", err.Error())
		os.Exit(5)
	}

	return allerrata
}

func ParseOval (data []byte) map[string]OvalData {
	// OvalDefinitions was generated 2019-04-24 22:06:30 by root on localhost.localdomain.
	type OvalDefinitions struct {
		XMLName        xml.Name `xml:"oval_definitions"`
		Text           string   `xml:",chardata"`
		Xmlns          string   `xml:"xmlns,attr"`
		Oval           string   `xml:"oval,attr"`
		RedDef         string   `xml:"red-def,attr"`
		UnixDef        string   `xml:"unix-def,attr"`
		Xsi            string   `xml:"xsi,attr"`
		SchemaLocation string   `xml:"schemaLocation,attr"`
		Definitions struct {
//			Text       string `xml:",chardata"`
			Definition []struct {
//				Text     string `xml:",chardata"`
//				Class    string `xml:"class,attr"`
				ID       string `xml:"id,attr"`
//				Version  string `xml:"version,attr"`
				Metadata struct {
//					Text     string `xml:",chardata"`
//					Title    string `xml:"title"`
//					Affected struct {
//						Text     string   `xml:",chardata"`
//						Family   string   `xml:"family,attr"`
//						Platform []string `xml:"platform"`
//					} `xml:"affected"`
					Reference []struct {
						Text   string `xml:",chardata"`
						RefID  string `xml:"ref_id,attr"`
//						RefURL string `xml:"ref_url,attr"`
//						Source string `xml:"source,attr"`
					} `xml:"reference"`
					Description string `xml:"description"`
					Advisory    struct {
						Text     string `xml:",chardata"`
//						From     string `xml:"from,attr"`
//						Severity string `xml:"severity"`
						Rights   string `xml:"rights"`
//						Issued   struct {
//							Text string `xml:",chardata"`
//							Date string `xml:"date,attr"`
//						} `xml:"issued"`
//						Updated struct {
//							Text string `xml:",chardata"`
//							Date string `xml:"date,attr"`
//						} `xml:"updated"`
						Cve []struct {
							Text   string `xml:",chardata"`
							Href   string `xml:"href,attr"`
							Public string `xml:"public,attr"`
							Impact string `xml:"impact,attr"`
							Cwe    string `xml:"cwe,attr"`
							Cvss2  string `xml:"cvss2,attr"`
							Cvss3  string `xml:"cvss3,attr"`
						} `xml:"cve"`
						Bugzilla []struct {
							Text string `xml:",chardata" xmlrpc:"summary"`
							Href string `xml:"href,attr" xmlrpc:"url"`
							ID   int64  `xml:"id,attr" xmlrpc:"id"`
						} `xml:"bugzilla"`
					} `xml:"advisory"`
				} `xml:"metadata"`
			} `xml:"definition"`
		} `xml:"definitions"`
	}

	var ovaldata OvalDefinitions
        _ = xml.Unmarshal([]byte(data), &ovaldata)
	oval := make(map[string]OvalData)

	for _, def := range ovaldata.Definitions.Definition {
		id := def.ID
		id = "CESA-" + id[len(id)-8:len(id)-4] + ":" + id[len(id)-4:]

		var cves []string
		var bugs []Bugzilla
		cvere, _ := regexp.Compile(`^CVE`)
		for _, ref := range def.Metadata.Reference {
			if cvere.MatchString(ref.RefID) {
				cves = append(cves, ref.RefID)
			}
		}
		for _, bug := range def.Metadata.Advisory.Bugzilla {
			bugs = append(bugs, bug)
		}

		var current = oval[id]
		current.Description = def.Metadata.Description
		current.Rights = def.Metadata.Advisory.Rights
		current.References = cves
		current.Bugs = bugs
		oval[id] = current
	}

	return oval
}

func get_packages_for_errata (errata Erratum, inv Inventory) []int64 {
	var pkglist []int64

	for _, rpm := range errata.Packages {
		if pkgid, ok := inv.filename2id[rpm]; ok {
			pkglist = append(pkglist, pkgid)
		}
	}

	return pkglist
}

func create_errata (client *xmlrpc.Client, sessionkey string, info SWerrata, bugs []Bugzilla, keywords []string, pkglist []int64, publish bool, channels []string) bool {
	params := make([]interface{}, 7)
	params[0] = sessionkey
	params[1] = info
	params[2] = bugs
	params[3] = keywords
	params[4] = pkglist
	params[5] = publish
	params[6] = channels

	type Response struct {
		Id			int64	`xmlrpc:"id"`
		Date			string	`xmlrpc:"date"`
		Advisory_Type		string	`xmlrpc:"advisory_type"`
		Advisory_Name		string	`xmlrpc:"advisory_name"`
		Advisory_Synopsis	string	`xmlrpc:"advisory_synopsis"`
	}

	var response Response
	err := client.Call("errata.create", params, &response)

	if err == nil && response.Id > 0 {
		return true
	}

	return false
}

func check_api_support (version string, supported []float64) bool {
	for _, i := range supported {
		if version == float_to_string(i) {
			return true
		}
	}

	return false
}

func include_channels (channels []string, include *[]string) []string {
	var result []string

	if len(*include) == 0 {
		return channels
	}

	for _, channel := range channels {
		var included bool = false
		for _, inc := range *include {
			if channel == inc {
				included = true
			}
		}
		if included {
			result = append(result, channel)
		}
	}

	return result
}

func exclude_channels (channels []string, exclude *[]string) []string {
	var result []string

	for _, channel := range channels {
		var excluded bool = false
		for _, exc := range *exclude {
			if channel == exc {
				excluded = true
			}
		}
		if !excluded {
			result = append(result, channel)
		}
	}

	return result
}

func float_to_string (input float64) string {
	if input == float64(int64(input)) {
		return fmt.Sprintf("%.0f", input)
	}

	return fmt.Sprintf("%.2f", input)
}

func string_to_float (input string) float64 {
	result, err := strconv.ParseFloat(input, 64)
	if err == nil {
		return result
	} else {
		return 0
	}
}

func add_issue_date (client *xmlrpc.Client, sessionkey string, errata string, issuedate time.Time) bool {
	type Details struct {
		IssueDate	time.Time	`xmlrpc:"issue_date"`
		UpdateDate	time.Time	`xmlrpc:"update_date"`
	}

	var details Details
	details.IssueDate = issuedate
	details.UpdateDate = issuedate

	params := make([]interface{}, 3)
	params[0] = sessionkey
	params[1] = errata
	params[2] = details

	var response int64
	err := client.Call("errata.set_details", params, &response)

	if err == nil && response > 0 {
		return true
	}

	return false
}

func add_severity (client *xmlrpc.Client, sessionkey string, errata string, severity string) bool {
	type Details struct {
		Severity	string	`xmlrpc:"severity"`
	}

	var details Details
	details.Severity = severity

	params := make([]interface{}, 3)
	params[0] = sessionkey
	params[1] = errata
        params[2] = details

	var response int64
	err := client.Call("errata.set_details", params, &response)

        if err == nil && response > 0 {
                return true
        }

        return false
}

func get_channels_of_packages (pkglist []int64, inv Inventory) []string {
	labels := make(map[string]bool)
	var result []string

	for _, pkg := range pkglist {
		for _, channel := range inv.id2channels[pkg] {
			labels[channel] = true
		}
	}

	for key := range labels {
		result = append(result, key)
	}

	return result
}

func publish_errata (client *xmlrpc.Client, sessionkey string, errata string, channels []string) bool {
        params := make([]interface{}, 3)
        params[0] = sessionkey
        params[1] = errata
        params[2] = channels

	type Response struct {
		Id			int	`xmlrpc:"id"`
		Date			string	`xmlrpc:"date"`
		AdvisoryType		string	`xmlrpc:"advisory_type"`
		AdvisoryName		string	`xmlrpc:"advisory_name"`
		AdvisorySynopsis	string	`xmlrpc:"advisory_synopsis"`
	}
	var response Response

	err := client.Call("errata.publish", params, &response)
	return err == nil
}

func add_cve_to_errata (client *xmlrpc.Client, sessionkey string, errata SWerrata, cves []string) bool {
	if cves == nil {
		// called without CVE information, so we bail nicely
		return true
	}

	type SWerrata2 struct {
		Synopsis	string	`xmlrpc:"synopsis"`
		AdvisoryName	string	`xmlrpc:"advisory_name"`
		AdvisoryRelease	int	`xmlrpc:"advisory_release"`
		AdvisoryType	string	`xmlrpc:"advisory_type"`
		From		string	`xmlrpc:"errataFrom"`
		Product		string	`xmlrpc:"product"`
		Topic		string	`xmlrpc:"topic"`
		Description	string	`xmlrpc:"description"`
		References	string	`xmlrpc:"references"`
		Notes		string	`xmlrpc:"notes"`
		Solution	string	`xmlrpc:"solution"`
		CVEs		[]string	`xmlrpc:"cves"`
	}

	var details SWerrata2
	details.Synopsis = errata.Synopsis
	details.AdvisoryName = errata.AdvisoryName
	details.AdvisoryRelease = errata.AdvisoryRelease
	details.AdvisoryType = errata.AdvisoryType
	details.From = errata.From
	details.Product = errata.Product
	details.Topic = errata.Topic
	details.Description = errata.Description
	details.References = errata.References
	details.Notes = errata.Notes
	details.Solution = errata.Solution
	details.CVEs = cves

	params := make([]interface{}, 3)
	params[0] = sessionkey
	params[1] = errata.AdvisoryName
        params[2] = details

	var response int64
	err := client.Call("errata.set_details", params, &response)

        if err == nil && response > 0 {
                return true
        }

        return false
}

func list_packages (client *xmlrpc.Client, sessionkey string, errata string) []int64 {
        params := make([]interface{}, 2)
        params[0] = sessionkey
        params[1] = errata

	type Response struct {
		Id	int64
	}

	var response []Response
	var result []int64

	err := client.Call("errata.list_packages", params, &response)
	if err != nil {
		return result
	}

	for _, pkg := range response {
		result = append(result, pkg.Id)
	}

	return result
}

func add_packages (client *xmlrpc.Client, sessionkey string, errata string, pkgs []int64) int64 {
        params := make([]interface{}, 3)
        params[0] = sessionkey
        params[1] = errata
        params[2] = pkgs

	var response int64
	err := client.Call("errata.add_packages", params, &response)
	if err != nil {
		return -1
	}

	return response
}

func errata_is_excluded (errata string, exerrata *[]string) bool {
	for _, excluded := range *exerrata {
		if errata == excluded {
			return true
		}
	}

	return false
}

func only_in_first_int64 (a []int64, b []int64) []int64 {
	var result []int64

	bmap := make(map[int64]bool)
	for _, value := range b {
		bmap[value] = true
	}

	for _, value := range a {
		if _, exists := bmap[value]; !exists {
			result = append(result, value)
		}
	}

	return result
}

func only_in_first_string (a []string, b []string) []string {
	var result []string

	bmap := make(map[string]bool)
	for _, value := range b {
		bmap[value] = true
	}

	for _, value := range a {
		if _, exists := bmap[value]; !exists {
			result = append(result, value)
		}
	}

	return result
}

func min_log_level (debug bool, quiet bool) string {
	if debug { return "DEBUG" }
	if quiet { return "ERROR" }
	return "INFO"
}

func download_bzip2 (url string) *[]byte {
	log.Printf("[DEBUG] Downloading %s", url)
	client := &http.Client{}
	request, _ := http.NewRequest("GET", url, nil)
	request.Header.Set("User-Agent", "errata-import/" + strconv.Itoa(Version))
	resp, err := client.Do(request)
        if err != nil {
		log.Printf("[ERROR] Download failed: %v", err.Error())
		return &[]byte{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[ERROR] Download failed with status code %d", resp.StatusCode)
		os.Exit(7)
		return &[]byte{}
	}

	data, err := ioutil.ReadAll(bzip2.NewReader(resp.Body))
	if err != nil {
		log.Printf("[ERROR] Decompressing failed: %v", err.Error())
		return &[]byte{}
	}

	return &data
}

func file_content (file string) *[]byte {
       if file == "" {
                return &[]byte{}
        }

        if _, err := os.Stat(file); os.IsNotExist(err) {
                return &[]byte{}
        }

        data, err := ioutil.ReadFile(file)
	if err != nil {
		return &[]byte{}
	}

        return &data
}

func list_providing_channels (client *xmlrpc.Client, sessionkey string, pkgid int64) []string {
        params := make([]interface{}, 2)
        params[0] = sessionkey
        params[1] = pkgid

	type Channel struct {
		Label		string	`xmlrpc:"label"`
		Parent_label	string	`xmlrpc:"parent_label"`
		Name		string	`xmlrpc:"name"`
	}
	var response []Channel
	var result []string
	err := client.Call("packages.list_providing_channels", params, &response)
	if err != nil {
		return result
	}

	for _, channel := range response {
		result = append(result, channel.Label)
	}

	return result
}

func remove_packages_from_channel (client *xmlrpc.Client, sessionkey string, channel string, pkgs []int64) bool {
        params := make([]interface{}, 3)
        params[0] = sessionkey
        params[1] = channel
        params[2] = pkgs

	var response int64
	err := client.Call("channel.software.remove_packages", params, &response)
        if err == nil && response > 0 {
                return true
        }

	return false
}

func running_as_root () bool {
	if syscall.Getuid() == 0  { return true }
	if syscall.Geteuid() == 0 { return true }
	return false
}

func get_channel_last_sync (client *xmlrpc.Client, sessionkey string, channel string) time.Time {
        params := make([]interface{}, 2)
        params[0] = sessionkey
        params[1] = channel

	type Response struct {
		Id		int		`xmlrpc:"id"`
		Name		string		`xmlrpc:"name"`
		LastSync	time.Time	`xmlrpc:"yumrepo_last_sync"`
	}

	var response Response
	err := client.Call("channel.software.get_details", params, &response)
        if err == nil {
                return response.LastSync
        }

	return time.Date(1970, 1, 1, 0, 0, 1, 0, time.UTC)
}

func channel_sync_repo (client *xmlrpc.Client, sessionkey string, channel string) bool {
        params := make([]interface{}, 2)
        params[0] = sessionkey
        params[1] = channel

	var response int64
	err := client.Call("channel.software.sync_repo", params, &response)
        return err == nil
}
