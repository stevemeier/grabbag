package main

import "bytes"
import "fmt"
import "log"
import "os/exec"
import "github.com/miekg/dns"
import "github.com/whiteShtef/clockwork"

type checkResult struct {
	qname	string
	qtype	string
	data	string
	success	bool
}

var resultChan chan checkResult
var DNSrequests chan *dns.Msg
var DNSresponses chan *dns.Msg

func main() {
        resultChan = make(chan checkResult, 20)
	DNSrequests = make(chan *dns.Msg, 20)
	DNSresponses = make(chan *dns.Msg, 20)

	go processor()
	sched := clockwork.NewScheduler()
//	sched.Schedule().Every(1).Minutes().Do( func() { run_check("/usr/bin/false") } )
	sched.Schedule().Every(15).Seconds().Do( func() { run_check("/bin/ls", []string{"/Users/smeier/tmp/foo"}) } )
	go sched.Run()

	dns.HandleFunc(".", handleDnsRequest)

	// start server
	port := 5300
	server := &dns.Server{Addr: "0.0.0.0:5300", Net: "udp"}
	log.Printf("Starting at %d\n", port)
	err := server.ListenAndServe()
	log.Println(err)
}

func run_check (command string, args []string) {
	_, rcode, err := sysexec(command, args, nil)
	if err == nil {
		resultChan <- checkResult{"foo.example.org.", "A", "1.2.3.4", rcode == 0}
	}
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

func processor () {
	dnsdata := make(map[string]map[string][]string)

	go func() {
	for latest := range resultChan {
		if latest.success {
			log.Printf("Adding: %s %s %s\n", latest.qname, latest.qtype, latest.data)
			// add record as test was successful
			if dnsdata[latest.qname] == nil {
				dnsdata[latest.qname] = make(map[string][]string)
			}
			dnsdata[latest.qname][latest.qtype] = append(dnsdata[latest.qname][latest.qtype], latest.data)
			dnsdata[latest.qname][latest.qtype] = makeUnique(dnsdata[latest.qname][latest.qtype])
		}
		if !latest.success {
			log.Printf("Deleting: %s %s %s\n", latest.qname, latest.qtype, latest.data)
			// delete record as test has failed
			if _, okn := dnsdata[latest.qname]; okn {
				if _, okt := dnsdata[latest.qname][latest.qtype]; okt {
					dnsdata[latest.qname][latest.qtype] = removeEntry(dnsdata[latest.qname][latest.qtype], latest.data)
				}
			}
		}
	}
}()

	go func() {
	for dnsq := range DNSrequests {
		answer := new(dns.Msg)
		answer.SetReply(dnsq)
		answer.Compress = false

		switch dnsq.Question[0].Qtype {
		case dns.TypeA:
			_, okn := dnsdata[dnsq.Question[0].Name]
			if okn {
				_, okt := dnsdata[dnsq.Question[0].Name]["A"]
				if okt {
					for _, ip4 := range dnsdata[dnsq.Question[0].Name]["A"] {
						rr, err := dns.NewRR(fmt.Sprintf("%s 60 IN A %s", dnsq.Question[0].Name, ip4))
						if err == nil {
							log.Println(rr)
							answer.Answer = append(answer.Answer, rr)
						} else {
							log.Println(err)
						}
					}
				}
			}

		case dns.TypeAAAA:
		default:
			answer.SetRcode(dnsq, 4)	// NOTIMP
		}

		if len(answer.Answer) == 0 {
			answer.SetRcode(dnsq, 2)	// SERVFAIL
		}

		DNSresponses <- answer
	}
}()
}

// https://cyruslab.net/2019/11/07/goremove-duplicate-elements-in-array-slice/
func makeUnique(names []string) []string {
    // if the name occurs the flag changes to true.
    // hence all name that has already occurred will be true.
    flag := make(map[string]bool)
    var uniqueNames []string
    for _, name := range names {
        if flag[name] == false {
            flag[name] = true
            uniqueNames = append(uniqueNames, name)
        }
    }
    // unique names collected
    return uniqueNames
}

func removeEntry(names []string, remove string) []string {
	var result []string
	for _, name := range names {
		if name == remove { continue }
		result = append(result, name)
	}

	return result
}

func handleDnsRequest (w dns.ResponseWriter, r *dns.Msg) {
	log.Printf("DNS request received: %s/%s\n", r.Question[0].Name, QtypeHumanReadable(r.Question[0].Qtype))
	DNSrequests <- r
	w.WriteMsg(<-DNSresponses)
	return
}

func QtypeHumanReadable (qtype uint16) (string) {
	switch qtype {
	case 1: return `A`
	case 28: return `AAAA`
	}
	return ``
}
