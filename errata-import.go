package main

import "encoding/json"
import "encoding/xml"
import "fmt"
import "io/ioutil"
import "github.com/DavidGamba/go-getoptions"
import "github.com/davecgh/go-spew/spew"
import "github.com/kolo/xmlrpc"
//import "github.com/sbabiv/xml2map"
import "log"
import "os"
import "regexp"
//import "strings"
import "strconv"
import "time"
//import "net"

// These two need to be loaded if cert-check is to be disabled
import "net/http"
import "crypto/tls"

const Version int = 20190426
const timelayout = "2006-01-02 15:04:05"
var SupportedAPI = []float64{10.9,  // Spacewalk 0.6
                             10.11, // Spacewalk 1.0 and 1.1
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
		    }

type Meta struct {
	Author		string
	Disclaimer	string
	License		string
	Timestamp	string
}

type Erratum struct {
	Id          string   `json:"id"`		// Only needed in array approach
	Description string   `json:"description"`
	From        string   `json:"from"`
	IssueDate   string   `json:"issue_date"`
	Manual      string   `json:"manual"`
	Notes       string   `json:"notes"`
	OsArch      []string `json:"os_arch"`
	OsRelease   []string `json:"os_release"`
	Packages    []string `json:"packages"`
	Product     string   `json:"product"`
	References  string   `json:"references"`
	Release     string   `json:"release"`
	Severity    string   `json:"severity"`
	Solution    string   `json:"solution"`
	Synopsis    string   `json:"synopsis"`
	Topic       string   `json:"topic"`
	Type        string   `json:"type"`
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

type Bug struct {
	Id		int
	Summary		string
	Url		string
}

type Inventory struct {
	filename2id	map[string]int64
	id2channels	map[int64][]string
}

type OvalData struct {
	Description	string
	References	[]string
	Rights		string
}

func main () {
	// This works if JSON data is an array
//	file, _ := ioutil.ReadFile("errata.test_array.json")
//	var errata []Erratum
//	_ = json.Unmarshal([]byte(file), &errata)
//	spew.Dump(errata)

	// This works if JSON data is a hash (as it currently is)
//	file, _ := ioutil.ReadFile("errata.test_map.json")
//	var errata = map[string]Erratum{}
//	_ = json.Unmarshal([]byte(file), &errata)
//	spew.Dump(errata)

	var debug bool
	var publish bool
	var server string

	var security bool
	var bugfix bool
	var enhancement bool

	var ignoreapiversion bool

	var inchannels *[]string
	var exchannels *[]string

	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	opt.StringVar(&server, "server", "localhost")
	opt.BoolVar(&publish, "publish", false)

	opt.BoolVar(&security, "security", false)
	opt.BoolVar(&bugfix, "bugfix", false)
	opt.BoolVar(&enhancement, "enhancement", false)

	opt.BoolVar(&ignoreapiversion, "ignore-api-version", false)

	inchannels = opt.StringSlice("include-channels", 1, 255)
	exchannels = opt.StringSlice("exclude-channels", 1, 255)

	remaining, err := opt.Parse(os.Args[1:])
	if err != nil {
		fmt.Println("Failed to parse options")
		os.Exit(4)
	}

	fmt.Printf("Remaining is %v\n", remaining)
	fmt.Printf("Debug is %t\n", debug)
	fmt.Printf("Publish is %t\n", publish)
	fmt.Printf("Server is %s\n", server)

	// If no errata type is selected, enable all
	if (!(security || bugfix || enhancement)) {
		security, bugfix, enhancement = true, true, true
	}

	// Test on a full dataset
//	file, _ := ioutil.ReadFile("/Users/smeier/tmp/errata.latest.json")
//	var allerrata = map[string]Erratum{}
//	_ = json.Unmarshal([]byte(file), &allerrata)
//	^^ works, but not with `meta` section
//	x := 1
//	spew.Dump(x)

	file, _ := ioutil.ReadFile("/Users/smeier/tmp/errata.newform.json")
	var allerrata Raw
	_ = json.Unmarshal([]byte(file), &allerrata)
//	spew.Dump(allerrata.Meta)

//	var home string = os.Getenv("HOME")
//	var latest map[string]interface{}

	// Test current XML format
//	if _, err := os.Stat(home + "/tmp/errata.latest.xml"); err == nil {
//		data, err := ioutil.ReadFile(home +"/tmp/errata.latest.xml")
//		if err != nil {
//			fmt.Println("Could not read " + home + "/tmp/errata.latest.xml")
//			os.Exit(1)
//		}
//		fmt.Println("Loading " + home + "/tmp/errata.latest.xml")
//		decoder := xml2map.NewDecoder(strings.NewReader(string(data[:])))
//		latest, err = decoder.Decode()
//		spew.Dump(latest)
//		_, err = decoder.Decode()
//	}
//	_ = latest


	// Load Red Hat OVAL data
	var oval map[string]OvalData = ParseOval("/Users/smeier/tmp/com.redhat.rhsa-all.xml")
	_ = oval

	// Disable TLS certificate checks (insecure!)
	// Source: https://stackoverflow.com/questions/12122159/how-to-do-a-https-request-with-bad-certificate
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Configure timeout
//	http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = time.Second * 5
//	^^ doesn't work

//	var myTransport http.RoundTripper = &http.Transport{
 //       Proxy:                 http.ProxyFromEnvironment,
  //      ResponseHeaderTimeout: time.Second * 5,
//	DialContext: (&net.Dialer{Timeout: time.Second * 5}).DialContext,
//	}
	// DialContext actually does it for unreachable server

	// Initialize XML-RPC Client
//	client, err := xmlrpc.NewClient("https://192.168.227.132/rpc/api", nil)
	client, err := xmlrpc.NewClient("https://" + server + "/rpc/api", nil)
//	client, err := xmlrpc.NewClient("https://" + server + "/rpc/api", myTransport)
//	^^ timeout is 5 minutes(!)
//	client, err := xmlrpc.NewClient("https://" + server + "/rpc/api", myTransport)
//	^^ should work
//	client, err := xmlrpc.NewClient("https://" + server + "/rpc/api", {Timeout: &timeout})
//	fmt.Fprintf(os.Stdout, "NewClient is type %T\n", client)
	if err != nil {
//		fmt.Println("Could not read XML")
		log.Fatal(err)
		os.Exit(2)
	}

	// Get server version
	var apiversion string
	client.Call("api.get_version", nil, &apiversion)
	spew.Dump(apiversion)

	if apiversion == "" {
		fmt.Println("Could not connect to server!");
		os.Exit(2)
	}

	if (!check_api_support(apiversion, SupportedAPI) && !ignoreapiversion) {
		fmt.Printf("API version %s is not supported!\n", apiversion)
		os.Exit(3)
	}

	username := "admin"
	password := "admin1"

	// Authenticate and get sessionKey
	var sessionkey string = init_session(client, username, password)
	if sessionkey == "" {
		fmt.Println("Authentication failed!")
		os.Exit(1)
	}

	// Check admin status
	if (user_is_admin(client, sessionkey, username)) {
		fmt.Printf("User %s has administrator access to this server\n", username)
	}

	// List all channels
	var channels []string = get_channel_list(client, sessionkey)
	fmt.Println("Full channel list:")
	spew.Dump(channels)

	fmt.Println("Include settings")
	spew.Dump(*inchannels)

	fmt.Println("Exclude settings")
	spew.Dump(*exchannels)

	channels = include_channels(channels, inchannels)
	channels = exclude_channels(channels, exchannels)

	fmt.Println("Filtered channel list:")
	spew.Dump(channels)

	// Get packages of channel
	var inv Inventory = get_inventory(client, sessionkey, channels)
	_ = inv
	fmt.Println("Server inventory:")
	spew.Dump(inv)

	fmt.Println("---")

	// Get existing errata
//	var existing = make(map[string]bool)
	var existing = get_existing_errata(client, sessionkey, channels)
	spew.Dump(existing)

//	fmt.Println("DATA from JSON:")
//	for _, errata := range allerrata {
//		for _, rpm := range errata.Packages {
//			fmt.Printf("%s includes package %s\n", errata.Id, rpm);
//		}
//	}
	// ^^ works
	fmt.Println("DATA from JSON:")
	for _, errata := range allerrata.Advisories {

		if (errata.Type == "Security Advisory" && !security) {
			fmt.Printf("Skipping %s\n", errata.Id)
			continue
		}
		if (errata.Type == "Bug Fix Advisory" && !bugfix) {
			fmt.Printf("Skipping %s\n", errata.Id)
			continue
		}
		if (errata.Type == "Product Enhancement Advisory" && !enhancement) {
			fmt.Printf("Skipping %s\n", errata.Id)
			continue
		}

		fmt.Printf("Processing %s\n", errata.Id)

		var pkglist []int64 = get_packages_for_errata(errata, inv)
		fmt.Println("Package ID list")
		spew.Dump(pkglist)

		if len(pkglist) == 0 {
			continue
		}

		var chanlist []string = get_channels_of_packages(pkglist, inv)
		fmt.Println("Channel label list")
		spew.Dump(chanlist)

		var success bool

		var info SWerrata
		info.AdvisoryName = errata.Id
		info.AdvisoryType = errata.Type
		info.Synopsis = errata.Synopsis
//		info.Description = errata.Description
		info.Description = get_oval_data(errata.Id, "Description", oval, errata.Description)
		info.Product = errata.Product
		info.References = errata.References
		info.Solution = errata.Solution
		info.Topic = errata.Topic
//		info.Notes = errata.Notes
		info.Notes = get_oval_data(errata.Id, "Rights", oval, errata.Notes)
		info.From = errata.From

		if exists := existing[(errata.Id)]; !exists {
			// Create Errata
			success = create_errata(client, sessionkey, info, []Bug{}, []string{}, pkglist, false, []string{})
			spew.Dump(success)
			if string_to_float(apiversion) >= 12 {
				fmt.Printf("Adding issue date to %s\n", errata.Id)
				issuedate, _ := time.Parse(timelayout, errata.IssueDate)
				success = add_issue_date(client, sessionkey, errata.Id, issuedate)
			}
			if string_to_float(apiversion) >= 21 && errata.Severity != "" {
				fmt.Printf("Adding severity %s to %s\n", errata.Severity, errata.Id)
				success = add_severity(client, sessionkey, errata.Id, errata.Severity)
			}
			if publish {
				fmt.Printf("Publishing %s\n", errata.Id)
				success = publish_errata(client, sessionkey, errata.Id, chanlist)
				if errata.Type == "Security Advisory" {
					fmt.Printf("Adding CVE information to %s\n", errata.Id)
					success = add_cve_to_errata(client, sessionkey, errata.Id, oval[(errata.Id)].References)
				}
			}
		} else {
			// Update Errata
			var curlist []int64 = list_packages(client, sessionkey, errata.Id)
			if len(pkglist) > len(curlist) {
				var pkgsadded int64 = add_packages(client, sessionkey, errata.Id, curlist)
				_ = pkgsadded
			}
		}


	}

	_ = close_session(client, sessionkey)
	os.Exit(0)
}

func init_session (client *xmlrpc.Client, username string, password string) string {
	params := make([]interface{}, 2)
	params[0] = username
	params[1] = password

	var sessionkey string
	client.Call("auth.login", params, &sessionkey)

	return sessionkey
}

func close_session (client *xmlrpc.Client, sessionkey string) bool {
	params := make([]interface{}, 1)
	params[0] = sessionkey

	client.Call("auth.logout", params, nil)

	return true
}

func user_is_admin (client *xmlrpc.Client, sessionkey string, username string) bool {
	params := make([]interface{}, 2)
	params[0] = sessionkey
	params[1] = username

	var roles []string
	client.Call("user.list_roles", params, &roles)

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
	client.Call("channel.list_all_channels", params, &channels)

	var channelnames []string
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
	for _, channel := range channels {
		params[0] = sessionkey
		params[1] = channel

		var packages []interface{}
		client.Call("channel.software.list_all_packages", params, &packages)

		for _, pkg := range packages {
			if details, ok := pkg.(map[string]interface{}); ok {
				id := details["id"].(int64)
				filename, inchannels := get_package_details(client, sessionkey, id)
				fmt.Printf("Adding %s (%d) to inventory\n", filename, id)
				inv.filename2id[filename] = id
				inv.id2channels[id] = inchannels
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
	fmt.Println("Fetching unpublished errata")
	client.Call("errata.list_unpublished_errata", params, &unpub)
//	spew.Dump(unpub)
	for _, errata := range unpub {
		existing[(errata.AdvisoryName)] = true
	}

	for _, channel := range channels {
		params[1] = channel
		fmt.Printf("Fetching existing errata for channel %s\n", channel)
//		var response []Response
		client.Call("channel.software.list_errata", params, &response)
//		spew.Dump(response)

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
	client.Call("packages.get_details", params, &details)

	if detail, ok := details.(map[string]interface{}); ok {
		for _, provchan := range detail["providing_channels"].([]interface{}) {
			inchannels = append(inchannels, provchan.(string))
		}
		return detail["file"].(string), inchannels
	}

	return "", []string{}
}

func ParseOval(file string) map[string]OvalData {
	if file == "" {
		return nil
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

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
		Generator      struct {
			Text           string `xml:",chardata"`
			ProductName    string `xml:"product_name"`
			ProductVersion string `xml:"product_version"`
			SchemaVersion  string `xml:"schema_version"`
			Timestamp      string `xml:"timestamp"`
			ContentVersion string `xml:"content_version"`
		} `xml:"generator"`
		Definitions struct {
			Text       string `xml:",chardata"`
			Definition []struct {
				Text     string `xml:",chardata"`
				Class    string `xml:"class,attr"`
				ID       string `xml:"id,attr"`
				Version  string `xml:"version,attr"`
				Metadata struct {
					Text     string `xml:",chardata"`
					Title    string `xml:"title"`
					Affected struct {
						Text     string   `xml:",chardata"`
						Family   string   `xml:"family,attr"`
						Platform []string `xml:"platform"`
					} `xml:"affected"`
					Reference []struct {
						Text   string `xml:",chardata"`
						RefID  string `xml:"ref_id,attr"`
						RefURL string `xml:"ref_url,attr"`
						Source string `xml:"source,attr"`
					} `xml:"reference"`
					Description string `xml:"description"`
					Advisory    struct {
						Text     string `xml:",chardata"`
						From     string `xml:"from,attr"`
						Severity string `xml:"severity"`
						Rights   string `xml:"rights"`
						Issued   struct {
							Text string `xml:",chardata"`
							Date string `xml:"date,attr"`
						} `xml:"issued"`
						Updated struct {
							Text string `xml:",chardata"`
							Date string `xml:"date,attr"`
						} `xml:"updated"`
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
							Text string `xml:",chardata"`
							Href string `xml:"href,attr"`
							ID   string `xml:"id,attr"`
						} `xml:"bugzilla"`
						AffectedCpeList struct {
							Text string   `xml:",chardata"`
							Cpe  []string `xml:"cpe"`
						} `xml:"affected_cpe_list"`
					} `xml:"advisory"`
				} `xml:"metadata"`
				Criteria struct {
					Text      string `xml:",chardata"`
					Operator  string `xml:"operator,attr"`
					Criterion []struct {
						Text    string `xml:",chardata"`
						Comment string `xml:"comment,attr"`
						TestRef string `xml:"test_ref,attr"`
					} `xml:"criterion"`
					Criteria []struct {
						Text      string `xml:",chardata"`
						Operator  string `xml:"operator,attr"`
						Criterion []struct {
							Text    string `xml:",chardata"`
							Comment string `xml:"comment,attr"`
							TestRef string `xml:"test_ref,attr"`
						} `xml:"criterion"`
						Criteria []struct {
							Text     string `xml:",chardata"`
							Operator string `xml:"operator,attr"`
							Criteria []struct {
								Text      string `xml:",chardata"`
								Operator  string `xml:"operator,attr"`
								Criterion []struct {
									Text    string `xml:",chardata"`
									Comment string `xml:"comment,attr"`
									TestRef string `xml:"test_ref,attr"`
								} `xml:"criterion"`
							} `xml:"criteria"`
							Criterion []struct {
								Text    string `xml:",chardata"`
								Comment string `xml:"comment,attr"`
								TestRef string `xml:"test_ref,attr"`
							} `xml:"criterion"`
						} `xml:"criteria"`
					} `xml:"criteria"`
				} `xml:"criteria"`
			} `xml:"definition"`
		} `xml:"definitions"`
		Tests struct {
			Text        string `xml:",chardata"`
			RpminfoTest []struct {
				Text    string `xml:",chardata"`
				Check   string `xml:"check,attr"`
				Comment string `xml:"comment,attr"`
				ID      string `xml:"id,attr"`
				Version string `xml:"version,attr"`
				Object  struct {
					Text      string `xml:",chardata"`
					ObjectRef string `xml:"object_ref,attr"`
				} `xml:"object"`
				State struct {
					Text     string `xml:",chardata"`
					StateRef string `xml:"state_ref,attr"`
				} `xml:"state"`
			} `xml:"rpminfo_test"`
		} `xml:"tests"`
		Objects struct {
			Text          string `xml:",chardata"`
			RpminfoObject []struct {
				Text    string `xml:",chardata"`
				ID      string `xml:"id,attr"`
				Version string `xml:"version,attr"`
				Name    string `xml:"name"`
			} `xml:"rpminfo_object"`
		} `xml:"objects"`
		States struct {
			Text         string `xml:",chardata"`
			RpminfoState []struct {
				Text           string `xml:",chardata"`
				ID             string `xml:"id,attr"`
				AttrVersion    string `xml:"version,attr"`
				SignatureKeyid struct {
					Text      string `xml:",chardata"`
					Operation string `xml:"operation,attr"`
				} `xml:"signature_keyid"`
				Version struct {
					Text      string `xml:",chardata"`
					Operation string `xml:"operation,attr"`
				} `xml:"version"`
				Arch struct {
					Text      string `xml:",chardata"`
					Datatype  string `xml:"datatype,attr"`
					Operation string `xml:"operation,attr"`
				} `xml:"arch"`
				Evr struct {
					Text      string `xml:",chardata"`
					Datatype  string `xml:"datatype,attr"`
					Operation string `xml:"operation,attr"`
				} `xml:"evr"`
			} `xml:"rpminfo_state"`
		} `xml:"states"`
	}

	var ovaldata OvalDefinitions
	data, _ := ioutil.ReadFile(file)
        _ = xml.Unmarshal([]byte(data), &ovaldata)
	oval := make(map[string]OvalData)

	for _, def := range ovaldata.Definitions.Definition {
		id := def.ID
		id = "CESA-" + id[len(id)-8:len(id)-4] + ":" + id[len(id)-4:]

		var cves []string
		cvere, _ := regexp.Compile(`^CVE`)
		for _, ref := range def.Metadata.Reference {
//			matched, _ := regexp.MatchString(`^CVE`, ref.RefID)
//			matched, _ := regexp.MatchString(cvere, ref.RefID)
//			matched, _ := cvere.MatchString(ref.RefID)
//			if matched {
			if cvere.MatchString(ref.RefID) {
				cves = append(cves, ref.RefID)
			}
		}

		var current = oval[id]
		current.Description = def.Metadata.Description
		current.Rights = def.Metadata.Advisory.Rights
		current.References = cves
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

func create_errata (client *xmlrpc.Client, sessionkey string, info SWerrata, bugs []Bug, keywords []string, pkglist []int64, publish bool, channels []string) bool {
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
	client.Call("errata.create", params, &response)

	if response.Id > 0 {
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

	var response int
	client.Call("errata.set_details", params, &response)

	if response > 0 {
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

	var response int
        client.Call("errata.set_details", params, &response)

        if response > 0 {
                return true
        }

        return false
}

func get_oval_data (errata string, field string, oval map[string]OvalData, unchanged string) string {
	if _, exists := oval[errata]; exists {
		if field == "Description" {
			if len(oval[errata].Description) > 4000 {
				return oval[errata].Description[:3999]
			} else {
				return oval[errata].Description
			}
		}
		if field == "Rights" {
			return "The description and CVE numbers have been taken from Red Hat OVAL definitions.\n\n" + oval[errata].Rights
		}
	}

	return unchanged
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

	var response int
        client.Call("errata.publish", params, &response)

	return true
}

func add_cve_to_errata (client *xmlrpc.Client, sessionkey string, errata string, cves []string) bool {
	type Details struct {
		cves	[]string	`xmlrpc:"cves"`
	}

	var details Details
	details.cves = cves

	params := make([]interface{}, 3)
	params[0] = sessionkey
	params[1] = errata
        params[2] = details

	var response int
        client.Call("errata.set_details", params, &response)

        if response > 0 {
                return true
        }

        return false
}

func set_compare (a []string, b []string, mode bool) bool {
	// if mode is false, check for subset
	// if mode is true, check for superset
	bmap := make(map[string]bool)

	for _, key := range b {
		bmap[key] = true
	}

	for _, key := range a {
		if exists := bmap[key]; exists {
			delete(bmap, key)
		}
	}

	if len(bmap) > 0 {
		// b has more elements than a
		return false || !mode
	}

	// a has more elements than b (or is identical)
	return true && mode
}

func list_packages (client *xmlrpc.Client, sessionkey string, errata string) []int64 {
        params := make([]interface{}, 2)
        params[0] = sessionkey
        params[1] = errata

	type Response struct {
		Id	int64
	}

	var response []Response

        client.Call("errata.list_packages", params, &response)

	var result []int64
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
	client.Call("errata.add_packages", params, &response)

	return response
}
