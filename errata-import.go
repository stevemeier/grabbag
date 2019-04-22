package main

import "encoding/json"
import "fmt"
import "io/ioutil"
import "github.com/DavidGamba/go-getoptions"
import "github.com/davecgh/go-spew/spew"
import "github.com/kolo/xmlrpc"
import "github.com/sbabiv/xml2map"
import "log"
import "os"
import "strings"
//import "time"
//import "net"

// These two need to be loaded if cert-check is to be disabled
import "net/http"
import "crypto/tls"

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
	Synopsis	string
	AdvisoryName	string
	AdvisoryRelease	int
	AdvisoryType	string
	Product		string
	Topic		string
	Description	string
	References	string
	Notes		string
	Solution	string
	Keyword		[]string
	Publish		bool
	ChannelLabel	[]string
}

type Inventory struct {
	filename2id	map[string]int64
	id2channels	map[int64][]string
}

// for "map"
//type Errata struct {
//	Id	string
//	Data	Erratum
//}

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
	var server string

	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	opt.StringVar(&server, "server", "localhost")

	remaining, err := opt.Parse(os.Args[1:])

	fmt.Printf("Remaining is %d\n", remaining)
	fmt.Printf("Debug is %t\n", debug)
	fmt.Printf("Server is %s\n", server)

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

	var home string = os.Getenv("HOME")
	var latest map[string]interface{}

	// Test current XML format
	if _, err := os.Stat(home + "/tmp/errata.latest.xml"); err == nil {
		data, err := ioutil.ReadFile(home +"/tmp/errata.latest.xml")
		if err != nil {
			fmt.Println("Could not read " + home + "/tmp/errata.latest.xml")
			os.Exit(1)
		}
		fmt.Println("Loading " + home + "/tmp/errata.latest.xml")
		decoder := xml2map.NewDecoder(strings.NewReader(string(data[:])))
		latest, err = decoder.Decode()
//		spew.Dump(latest)
		_, err = decoder.Decode()
	}


	// Load Red Hat OVAL data
	if _, err := os.Stat("~/tmp/com.redhat.rhsa-all.xml"); err == nil {
	        data, err := ioutil.ReadFile("~/tmp/com.redhat.rhsa-all.xml")
		if err != nil {
			fmt.Println("Could not read ~/tmp/com.redhat.rhsa-all.xml")
			os.Exit(1)
		}
		fmt.Println("Loading ~/tmp/com.redhat.rhsa-all.xml")
		decoder := xml2map.NewDecoder(strings.NewReader(string(data[:])))
		result, err := decoder.Decode()
		spew.Dump(result)
	}

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
	var version string
	client.Call("api.getVersion", nil, &version)
	spew.Dump(version)

	if version == "" {
		fmt.Println("Could not connect to server!");
		os.Exit(2)
	}

	username := "admin"
	password := "admin1"

	// Authenticate and get sessionKey
	var sessionkey string
//	params := make([]interface{}, 2)
//	params[0] = "admin"
//	params[1] = "admin1"

//	client.Call("auth.login", params, &sessionkey)
	sessionkey = init_session(client, username, password)
//	spew.Dump(sessionkey)

	// Get user roles
//	var roles []string
//	params = make([]interface{}, 2)
//	params[0] = sessionkey
//	params[1] = "admin"
//	client.Call("user.list_roles", params, &roles)
//	spew.Dump(roles)

	// Check admin status
	if (user_is_admin(client, sessionkey, username)) {
		fmt.Printf("User %s has administrator access to this server\n", username)
	}

	// List all channels
//	var channels []interface{}
//	params = make([]interface{}, 1)
//	params[0] = sessionkey
//	client.Call("channel.list_all_channels", params, &channels)
	var channels []string
	channels = get_channel_list(client, sessionkey)
	fmt.Println("Channel list:\n")
	spew.Dump(channels)

	// Get packages of channel
//	var packages []interface{}
//	params = make([]interface{}, 2)
//	params[0] = sessionkey
//	params[1] = "centos7-x86_64-centosplus"
//	client.Call("channel.software.list_all_packages", params, &packages)
//	spew.Dump(packages)
	var inv Inventory
	inv = get_inventory(client, sessionkey, channels)

	fmt.Println("---")
	spew.Dump(inv)

//	fmt.Println("DATA from JSON:")
//	for _, errata := range allerrata {
//		for _, rpm := range errata.Packages {
//			fmt.Printf("%s includes package %s\n", errata.Id, rpm);
//		}
//	}
	// ^^ works
	fmt.Println("DATA from JSON:")
	for _, errata := range allerrata.Advisories {
		for _, rpm := range errata.Packages {
			fmt.Printf("%s includes package %s\n", errata.Id, rpm);
		}
	}

	fmt.Println("DATA from XML:")
	for _, errata := range latest {
		spew.Dump(errata)
	}

	os.Exit(0)

	// Source: https://stackoverflow.com/a/31816267/1592267
//	for _, record := range packages {
//		log.Printf(" [===>] Record: %s", record)

//		if rec, ok := record.(map[string]interface{}); ok {
//			for key, val := range rec {
//				log.Printf(" [========>] %s = %s", key, val)
//				if (key == "name") {
//					fmt.Printf("%s -> %s", key, val)
//				}
//				if (key == "id") {
//					var packagedetails interface{}
//					params = make([]interface{}, 2)
//					params[0] = sessionkey
//					params[1] = val
//					fmt.Printf("\nGetting details for package %d\n", val)
//					client.Call("packages.get_details", params, &packagedetails)
//					spew.Dump(packagedetails)

//					if detail, ok := packagedetails.(map[string]interface{}); ok {
//						for dkey, dval := range detail {
//							fmt.Printf("%s -> %s\n", dkey, dval)
//						}
//					}
//				}
//			}
//		} else {
//			fmt.Printf("record not a map[string]interface{}: %v\n", record)
//		}
//	}
}

func init_session (client *xmlrpc.Client, username string, password string) string {
	params := make([]interface{}, 2)
	params[0] = username
	params[1] = password

	var sessionkey string
	client.Call("auth.login", params, &sessionkey)

	return sessionkey
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
