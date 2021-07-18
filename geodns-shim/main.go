package main

import "fmt"
import "strings"
import "github.com/miekg/dns"

func main() {
  fmt.Println(add_label(`www.lordy.de`, `de`, 1))
  fmt.Println(expand_label(`www.lordy.de`, `-de`, true, 0))
  fmt.Println(expand_label(`www.lordy.de`, `de-`, false, 0))
  dns.HandleFunc(".", handleDnsRequest)

  udpserver := &dns.Server{Addr: "127.0.0.1:5300", Net: "udp"}
  tcpserver := &dns.Server{Addr: "127.0.0.1:5300", Net: "tcp"}
  go func(){ _ = udpserver.ListenAndServe() }()
  go func(){ _ = tcpserver.ListenAndServe() }()

  // Don't exit
  select {}
}

func handleDnsRequest(w dns.ResponseWriter, r *dns.Msg) {

  var qname = extract_qname(r)
  var queryid = r.Id
  fmt.Printf("Received a query: %s\n", qname)

  var newqname = "time.lordy.de."
  var newq dns.Msg
  newq.Question = make([]dns.Question, 1)
  newq.Question[0] = dns.Question{newqname, dns.TypeA, dns.ClassINET}

  in, _ := dns.Exchange(&newq, "82.149.225.149:53")
  in.Question = r.Question

  in.Id = queryid
  for i, answer := range in.Answer {
    in.Answer[i], _ = dns.NewRR(strings.Replace(answer.String(), newqname, qname, -1))
  }

  w.WriteMsg(in)
}

func extract_qname (r *dns.Msg) (string) {
  return r.Question[0].Name
}

func add_label (name string, extra string, pos int) (string) {
  // Split DNS name into labels
  labels := strings.Split(name, `.`)

  // Slice for results
  var result []string

  // Copy labels step-by-step to results
  for i, label := range labels {
    if i == pos {
      result = append(result, extra)
    }
    result = append(result, label)
  }

  return strings.Join(result, `.`)
}

func expand_label (name string, extra string, suffix bool, pos int) (string) {
  // Split DNS name into labels
  labels := strings.Split(name, `.`)

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

  return strings.Join(result, `.`)
}
