package main

import "encoding/json"
import "fmt"
import "io/ioutil"
import "github.com/davecgh/go-spew/spew"
import "github.com/kolo/xmlrpc"
import "github.com/sbabiv/xml2map"
import "log"
import "os"
import "strings"

// These two need to be loaded if cert-check is to be disabled
import "net/http"
import "crypto/tls"

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

	// Test on a full dataset
	file, _ := ioutil.ReadFile("/Users/smeier/tmp/errata.latest.json")
	var errata = map[string]Erratum{}
	_ = json.Unmarshal([]byte(file), &errata)
//	x := 1
//	spew.Dump(x)

	// Load Red Hat OVAL data
	if _, err := os.Stat("com.redhat.rhsa-all.xml"); err == nil {
	        data, err := ioutil.ReadFile("com.redhat.rhsa-all.xml")
		if err != nil {
			fmt.Println("Could not read XML")
			os.Exit(1)
		}
		decoder := xml2map.NewDecoder(strings.NewReader(string(data[:])))
		result, err := decoder.Decode()
		spew.Dump(result)
	}

	// Initialize XML-RPC Client
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client, err := xmlrpc.NewClient("https://165.227.141.163/rpc/api", nil)
	if err != nil {
//		fmt.Println("Could not read XML")
		log.Fatal(err)
		os.Exit(2)
	}

	// Get server version
	var version string
	client.Call("api.getVersion", nil, &version)
	spew.Dump(version)

	// Authenticate and get sessionKey
	var sessionkey string
	params := make([]interface{}, 2)
	params[0] = "admin"
	params[1] = "admin1"

	client.Call("auth.login", params, &sessionkey)
	spew.Dump(sessionkey)

	// Get user roles
	var roles []string
	params = make([]interface{}, 2)
	params[0] = sessionkey
	params[1] = "admin"
	client.Call("user.list_roles", params, &roles)
	spew.Dump(roles)

	// List all channels
	var channels []interface{}
	params = make([]interface{}, 1)
	params[0] = sessionkey
	client.Call("channel.list_all_channels", params, &channels)
	spew.Dump(channels)

	// Get packages of channel
	var packages []interface{}
	params = make([]interface{}, 2)
	params[0] = sessionkey
	params[1] = "centos7-x86_64-centosplus"
	client.Call("channel.software.list_all_packages", params, &packages)
	spew.Dump(packages)
}
