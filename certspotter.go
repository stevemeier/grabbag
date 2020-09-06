package main

import "crypto/sha256"
import "crypto/x509"
import "encoding/pem"
import "encoding/json"
import "fmt"
import "io/ioutil"
import "log"
import "os"
import "net/http"
import "time"

//import "github.com/davecgh/go-spew/spew"
import "github.com/DavidGamba/go-getoptions"

type SearchResult struct {
	ID           string   `json:"id"`
	TbsSha256    string   `json:"tbs_sha256"`
	DNSNames     []string `json:"dns_names"`
	PubkeySha256 string   `json:"pubkey_sha256"`
	NotBefore    time.Time   `json:"not_before"`
	NotAfter     time.Time   `json:"not_after"`
	Cert         struct {
	//	Type   string `json:"type"`	/* unused */
		Sha256 string `json:"sha256"`
		Data   string `json:"data"`
	} `json:"cert"`
}

func main () {
	// Parse arguments
	opt := getoptions.New()

	var cert string
	var debug bool
	opt.StringVar(&cert, "cert", "", opt.Required())
        opt.BoolVar(&debug, "debug", false)
	remaining, err := opt.Parse(os.Args[1:])

	// Handle empty or unknown options
	if len(os.Args[1:]) == 0 {
		log.Print(opt.Help())
		os.Exit(1)
        }
	if err != nil {
		log.Fatalf("Could not parse options: %s\n", err)
		os.Exit(1)
	}
	if len(remaining) > 0 {
		log.Fatalf("Unsupported parameter: %s\n", remaining)
		os.Exit(1)
	}

	// read username and password from ENV
	token := os.Getenv("SSLMATE_APIKEY")
	if token == "" {
		log.Fatal("Please set $SSLMATE_APIKEY to your API key\n")
		os.Exit(1)
	}

	// Parse certificate
	subject, notbefore, certsha256, pubkeysha256 := get_cert_details(cert)
	if debug {
		fmt.Printf("Current cert subject:     %s\n", subject)
		fmt.Printf("Current cert start date:  %s\n", notbefore)
		fmt.Printf("Current cert fingerprint: %s\n", certsha256)
	}

	// Create HTTP client
	client := &http.Client{}

	// Search for matching certificates
	req, _ := http.NewRequest("GET", "https://api.certspotter.com/v1/issuances?match_wildcards=true&expand=cert&expand=dns_names&domain=" + subject, nil)
	req.Header.Add("Authorization", "Bearer " + token)
	resp, err := client.Do(req)

	if resp.StatusCode >= 400 || err != nil {
		log.Fatalf("API search failed with response: %s\n", resp.Status)
		os.Exit(3)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read API response: %s\n", err)
	}

	var searchresult []SearchResult
	err = json.Unmarshal(body, &searchresult)
	if err != nil {
		log.Fatalf("Could not parse search result: %s\n", err)
		os.Exit(3)
	}

	var bestcandidate SearchResult
	bestcandidate.NotBefore = time.Now()
	for _, certificate := range searchresult {
		if certificate.Cert.Sha256 == certsha256 {
			if debug {
				fmt.Printf("Found current certificate (%s) in search results\n", certsha256)
			}
			continue
		}

		if (certificate.NotBefore).After(notbefore) && (certificate.NotBefore).Before(bestcandidate.NotBefore) {
			if pubkeysha256 == certificate.PubkeySha256 {
				if debug {
					fmt.Printf("Found better candidate (signed %s)", certificate.NotBefore)
				}
				bestcandidate = certificate
			} else {
				if debug {
					fmt.Println("Found better candidate (signed %s) but with different public key. Ignoring it.", certificate.NotBefore)
				}
			}
		}
	}

	if bestcandidate.Cert.Data == "" {
		log.Fatal("No new certificate found\n");
		os.Exit(1)
	}

	// Write out new certificate
	crt, err := os.Create(cert)
	if err != nil {
		log.Fatalf("Could not open certificate file for writing: %s\n", err)
		os.Exit(4)
	}
	defer crt.Close()

	_, _ = crt.WriteString(format_certificate(bestcandidate.Cert.Data, 64))
	_ = crt.Sync()

	os.Exit(0)
}

func get_cert_details (filename string) (string, time.Time, string, string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read file: %s\n", filename)
		os.Exit(2)
	}

	block, _ := pem.Decode(file)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("Could not parse certificate: %s\n", err)
		os.Exit(2)
	}

	return (cert.Subject).String(), cert.NotBefore, fmt.Sprintf("%x", sha256.Sum256(cert.Raw)), fmt.Sprintf("%x", sha256.Sum256(cert.RawSubjectPublicKeyInfo))
}

func format_certificate (raw string, limit int) (string) {
	var result string

	result += "-----BEGIN CERTIFICATE-----\n"

	// https://www.socketloop.com/tutorials/golang-chunk-split-or-divide-a-string-into-smaller-chunk-example
	var charSlice []rune
	for _, char := range raw {
		charSlice = append(charSlice, char)
	}

	for len(charSlice) >= 1 {
		result += string(charSlice[:limit]) + "\n"
		charSlice = charSlice[limit:]

		if len(charSlice) < limit {
			limit = len(charSlice)
		}
	}

	result += "-----END CERTIFICATE-----\n"

	return result
}
