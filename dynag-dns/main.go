package main

import "bytes"
import "log"
import "os/exec"
import "strconv"
import "strings"
import "time"
import "github.com/gofrs/uuid"
import "github.com/miekg/dns"
import "github.com/whiteShtef/clockwork"
import config "github.com/olebedev/config"

type checkResult struct {
	Uuid	string
	Success	bool
}

type DynRR struct {
	Data		dns.RR
	Enabled		bool
	Uuid		string
	LastChange	time.Time
}

var resultChan chan checkResult
var DNSrequests chan *dns.Msg
var DNSresponses chan *dns.Msg

func main() {
	// Initialize channels
        resultChan = make(chan checkResult, 20)
	DNSrequests = make(chan *dns.Msg, 20)
	DNSresponses = make(chan *dns.Msg, 20)

	// Initialize DNS data structure
	dnsdata := make(map[string]map[uint16][]*DynRR)
	// First level is `qname`, e.g. www.example.org.
	// Second level is `qtype`, e.g. 1 for A or 28 for AAAA
	// Last level is an array of records under this name and type

	// Load config file
	cfg, conferr := LoadConfig("./config.json")
	if conferr != nil {
		log.Fatal(conferr)
	}

	// Set up the scheduler
	sched := clockwork.NewScheduler()

	// Iterate over names in configuration
	for i := 0; i < len(cfg.UList("names")); i++ {
		uuid := GenerateUUID()
		nameconf, _ := cfg.Get("names." + strconv.Itoa(i))
		newrr, _ := dns.NewRR(nameconf.UString("name") + " " + nameconf.UString("rr"))
		header := newrr.Header()

		// Add name to the DNS data structure
		dnsdata[header.Name] = make(map[uint16][]*DynRR)
		dnsdata[header.Name][header.Rrtype] = append(dnsdata[header.Name][header.Rrtype], &DynRR{Data: newrr, Enabled: false, Uuid: uuid})

		// Schedule checks for this entry
		sched.Schedule().Every(nameconf.UInt("interval")).Seconds().Do( func() { run_check(nameconf.UString("command"), uuid) })
	}

	// Start the processing of check results and DNS queries
	go processor(dnsdata)

	// Run the previously configured scheduler
	go sched.Run()

	// Register the DNS handler
	dns.HandleFunc(".", handleDnsRequest)

	// Set up the server objects
	listenon := cfg.UString("server.listen","127.0.0.1") + ":" + cfg.UString("server.port","53")
	log.Printf("Listening on %s", listenon)
	udpserver := &dns.Server{Addr: listenon, Net: "udp"}
	tcpserver := &dns.Server{Addr: listenon, Net: "tcp"}

	// Start listeners
	go func(){ _ = udpserver.ListenAndServe() }()
	go func(){ _ = tcpserver.ListenAndServe() }()

	// Don't exit
	select {}
}

func run_check (command string, uuid string) {
	// Split the command into it's first bit (the binary) and the rest (the arguments)
	comsplit := strings.Split(command, ` `)

	// Call the sysexec function to run the check (nil == no input to the called binary)
	_, rcode, _ := sysexec(comsplit[0], comsplit[1:], nil)

	// Put the return code of the check into the results channel
	resultChan <- checkResult{Uuid: uuid, Success: rcode == 0}

	return
}

func sysexec (command string, args []string, input []byte) ([]byte, int, error) {
        var output bytes.Buffer

        cmd := exec.Command(command, args...)
        cmd.Stdin = bytes.NewBuffer(input)
        cmd.Stdout = &output
        err := cmd.Run()

        exitcode := 0
        if exitError, ok := err.(*exec.ExitError); ok {
                exitcode = exitError.ExitCode()
        }

        return output.Bytes(), exitcode, err
}

func processor (dnsdata map[string]map[uint16][]*DynRR) {
	go func() {
	// Process check results
	for check := range resultChan {
		// Find the record this check result applies to
		// Inefficient, but this is a proof of concept
		for qname, _ := range dnsdata {
			for qtype, _ := range dnsdata[qname] {
				for _, record := range dnsdata[qname][qtype] {
					if record.Uuid == check.Uuid && record.Enabled != check.Success {
						// Change the boolean status, log it, and set a timestamp (for debugging)
						log.Printf("Status change for %s to %s\n",
						record.Data.String(),
						UpBool(check.Success))

						record.Enabled = check.Success
						record.LastChange = time.Now()
					}
				}
			}
		}
	}
	}()

	go func() {
	// Process DNS queries
	for query := range DNSrequests {
		// Setup an answer object
		answer := new(dns.Msg)
		answer.Compress = false

		// Handle non INET class (respond with NOTIMP)
		if query.Question[0].Qclass != dns.ClassINET {
			answer.SetRcode(query, dns.StringToRcode["NOTIMP"])
			DNSresponses <- answer
			continue
		}

		// If this record is not in the map, we return NXDOMAIN
		if len(dnsdata[query.Question[0].Name]) == 0 {
			answer.SetRcode(query, dns.StringToRcode["NXDOMAIN"])
			DNSresponses <- answer
			continue
		}

		// If no record for this type exists, we should return empty NOERROR
		if len(dnsdata[query.Question[0].Name][query.Question[0].Qtype]) == 0 {
			answer.SetRcode(query, dns.StringToRcode["NOERROR"])
			DNSresponses <- answer
			continue
		}

		// We need to check if at least one record is `Enabled`
		// If there is none, return SERVFAIL, to prevent caching
		enabled := 0
		for _, record := range dnsdata[query.Question[0].Name][query.Question[0].Qtype] {
			if record.Enabled { enabled++ }
		}

		if enabled == 0 {
			answer.SetRcode(query, dns.StringToRcode["SERVFAIL"])
			DNSresponses <- answer
			continue
		}

		// At this point, data should be available, so we construct a proper answer
		answer.SetReply(query)
		for _, record := range dnsdata[query.Question[0].Name][query.Question[0].Qtype] {
			if record.Enabled {
				answer.Answer = append(answer.Answer, record.Data)
			}
		}

		// Send our compiled answer
		DNSresponses <- answer
	}
	}()
}

func handleDnsRequest (w dns.ResponseWriter, r *dns.Msg) {
	// Log
	log.Printf("DNS request received: %s/%s from %s\n", r.Question[0].Name, dns.TypeToString[r.Question[0].Qtype], w.RemoteAddr())

	// Put the request into the DNSrequests channel
	DNSrequests <- r

	// Write the latest response from the DNSresponses channel to the network
	werr := w.WriteMsg(<-DNSresponses)
	if werr != nil {
		log.Println(werr)
	}

	return
}

func GenerateUUID () string {
	// Generates a temporary UUID (only used at runtime, not stored or configurable)
	result, _ := uuid.NewV4()
	return result.String()
}

func LoadConfig (path string) (*config.Config, error) {
  cfg, err := config.ParseJsonFile(path)
  return cfg, err
}

func UpBool (in bool) string {
  return strings.ToUpper(strconv.FormatBool(in))
}
