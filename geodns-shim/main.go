package main

import "fmt"
import "log"
import "net"
import "os"
import "regexp"
import "strconv"
import "strings"
import "github.com/miekg/dns"
import "github.com/oschwald/geoip2-golang"
import "github.com/DavidGamba/go-getoptions"

// Global variables, used in the DNS handling function
var geodb *geoip2.Reader
var dnsbackend string
var nxfallback bool
var debug bool
var rewrite map[string]string

func main() {
  // Define parameters
  var listen string
  var dnsport string
  var geodbfile string
  opt := getoptions.New()
  opt.StringVar(&geodbfile, "geodb", "", opt.Required())
  opt.StringVar(&dnsbackend, "backend", "", opt.Required())
  opt.StringVar(&listen, "listen", "127.0.0.1")
  opt.StringVar(&dnsport, "port", "53")
  opt.BoolVar(&nxfallback, "nxfallback", false)
  opt.BoolVar(&debug, "debug", false)
  opt.StringMapVar(&rewrite, "rewrite", 2, 3, opt.Required())

  // Parse parameters
  remaining, err := opt.Parse(os.Args[1:])

  // Handle empty or unknown options
  if len(os.Args[1:]) == 0 {
    fmt.Print(opt.Help())
    os.Exit(1)
  }
  // Handle option parsing error
  if err != nil {
    fmt.Printf("Could not parse options: %s\n", err)
    os.Exit(1)
  }
  if len(remaining) > 0 {
    fmt.Printf("Unsupported parameter: %s\n", remaining)
    os.Exit(1)
  }

  // Backend needs to contain a port number (default :53)
  hasport, _ := regexp.MatchString(`:`, dnsbackend)
  if !hasport { dnsbackend = dnsbackend + `:53` }

  // Open GeoIP database
  geodb, _ = geoip2.Open(geodbfile)

  // Register DNS handler
  dns.HandleFunc(".", handleDnsRequest)

  // Define server paramters
  udpserver := &dns.Server{Addr: listen + `:` + dnsport, Net: "udp"}
  tcpserver := &dns.Server{Addr: listen + `:` + dnsport, Net: "tcp"}

  // Run servers in anonymous functions (permanently)
  go func(){ _ = udpserver.ListenAndServe() }()
  go func(){ _ = tcpserver.ListenAndServe() }()

  // Don't exit
  select {}
}

func handleDnsRequest (w dns.ResponseWriter, r *dns.Msg) {
  // Catch classes other than IN
  if r.Question[0].Qclass != dns.ClassINET {
    answer := new(dns.Msg)
    answer.SetRcode(r, dns.StringToRcode["NOTIMP"])
    werr := w.WriteMsg(answer)
    if werr != nil { log.Println(werr) }
  }

  // Use Client IP to get location (as ISO code)
  isocode := get_ip_location(w.RemoteAddr().String())

  // Store the client's Query ID for later use in the answer
  var queryid = r.Id

  // Extract the qname (e.g. www.example.org) from the DNS packet
  var qname = r.Question[0].Name

  if debug {
    log.Printf("Received query: %s/%s %s/%s\n", qname,
                                                dns.TypeToString[r.Question[0].Qtype],
						w.RemoteAddr().String(),
						isocode)
  }

  // Transform the query
  posint, _ := strconv.Atoi(rewrite["pos"])
  var newqname string
  switch rewrite["mode"] {
    case "add":
      newqname = add_label(qname, isocode, posint)
    case "suffix":
      newqname = expand_label(qname, rewrite["sep"] + isocode, true, posint)
    case "prefix":
      newqname = expand_label(qname, isocode + rewrite["sep"], false, posint)
    default:
      newqname = add_label(qname, isocode, 1)
  }
  if debug { log.Printf("Rewriting %s -> %s\n", qname, newqname) }

  // Send a query for the transformed name to the backend
  var newq dns.Msg
  newq.Question = make([]dns.Question, 1)
  newq.Question[0] = dns.Question{Name: newqname, Qtype: dns.TypeA, Qclass: dns.ClassINET}

  in, _ := dns.Exchange(&newq, dnsbackend)
  if nxfallback && in.Rcode == 3 {
    if debug { log.Printf("Got NXDOMAIN for %s, resending with original qname %s\n", newqname, qname) }
    in, _ = dns.Exchange(r, dnsbackend)
  }
  in.Question = r.Question

  // Put the original Query ID into the answer we got from the server
  in.Id = queryid
  for i, answer := range in.Answer {
    // Rewrite the answer we got from the backend to fit the client's original query
    in.Answer[i], _ = dns.NewRR(strings.Replace(answer.String(), newqname, qname, -1))
  }

  // Send it!
  werr := w.WriteMsg(in)
  if werr != nil { log.Println(werr) }
}

func add_label (name string, extra string, pos int) (string) {
  // Split DNS name into labels
  labels := dns.SplitDomainName(name)

  // Slice for results
  var result []string

  // Copy labels step-by-step to results
  for i, label := range labels {
    if i == pos {
      result = append(result, extra)
    }
    result = append(result, label)
  }

  return dns.Fqdn(strings.Join(result, `.`))
}

func expand_label (name string, extra string, suffix bool, pos int) (string) {
  // Split DNS name into labels
  labels := dns.SplitDomainName(name)

  // Slice for results
  var result []string

  // Copy labels step-by-step to results
  for i, label := range labels {
    if i == pos {
      if suffix {
        result = append(result, (label + extra))
      } else {
	// as prefix
        result = append(result, (extra + label))
      }

    } else {
      result = append(result, label)
    }
  }

  return dns.Fqdn(strings.Join(result, `.`))
}

func get_ip_location (ip string) (string) {
  // Input is host:port (e.g. 127.0.0.1:53462), but we need the IP address only
  host, _, _ := net.SplitHostPort(ip)

  // Lookup the IP address in the database
  record, err := geodb.City(net.ParseIP(host))
  if err != nil { log.Printf("Lookup failed for %s: %s\n", ip, err.Error()); return `` }

  return strings.ToLower(record.Country.IsoCode)
}
