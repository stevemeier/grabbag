package main

import "errors"
import "fmt"
import "os"
import "strings"
import "time"
import "github.com/DavidGamba/go-getoptions"
import "github.com/miekg/dns"
import "github.com/olorin/nagiosplugin"

// based on https://github.com/fibbs/nagios-check_dnsbl/blob/master/check_dnsbl.sh

var dnsserver string

func main() {
	var host string
	var lists []string
	var printlists bool
	opt := getoptions.New()
	opt.StringVar(&host, "host", "", opt.Alias("H"))
	opt.StringSliceVar(&lists, "lists", 1, 99, opt.Alias("l"))
	opt.BoolVar(&printlists, "p", false)
	opt.StringVar(&dnsserver, "dns", "", opt.Alias("s"))
	opt.Parse(os.Args[1:])

	if len(lists) == 0 {
		lists = []string{"ix.dnsbl.manitu.net",
	                         "dnsbl.sorbs.net",
	                         "bl.spamcop.net",
	                         "sbl.spamhaus.org",
	                         "xbl.spamhaus.org",
	                         "pbl.spamhaus.org",
	                         "dnsbl-1.uceprotect.net",
	                         "psbl.surriel.com",
	                         "l2.apews.org",
	                         "dnsrbl.swinog.ch",
	                         }
	}

	if printlists {
		for _, list := range lists {
			fmt.Println(list)
		}
		os.Exit(0)
	}

	if len(host) == 0 {
		fmt.Println(opt.Help())
		os.Exit(1)
	}

	check := nagiosplugin.NewCheck()
	defer check.Finish()

	// Read /etc/resolv.conf if no DNS server was set
	if len(dnsserver) == 0 {
		dnsconfig, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
		if len(dnsconfig.Servers) == 0 {
			check.AddResult(nagiosplugin.UNKNOWN, "No DNS server found in /etc/resolv.conf")
			check.Finish()
		}

		dnsserver = dnsconfig.Servers[0]
	}

	var listed_in []string
	for _, list := range lists {
		found, lookuperr := is_listed(host, list)
		if lookuperr != nil {
			check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Error looking up %s: %s", list, lookuperr.Error()))
			check.Finish()
		}
		if found {
			listed_in = append(listed_in, list)
		}

		time.Sleep(50 * time.Millisecond)
	}

	if len(listed_in) > 0 {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("%s is listed in %d DNSBLs (%s)", host, len(listed_in), strings.Join(listed_in, ",")))
		check.Finish()
	} else {
		check.AddResult(nagiosplugin.OK, fmt.Sprintf("%s is not on any of %d DNSBLs", host, len(lists)))
		check.Finish()
	}
}

func reverse_ip (s string) (string) {
	// https://medium.com/@tzuni_eh/go-append-prepend-item-into-slice-a4bf167eb7af
	var dummy []string
	for _, i := range strings.Split(s, ".") {
		dummy = append([]string{i}, dummy...)
	}

	return strings.Join(dummy, ".")
}

func has_dns_entry (s string) (bool, error) {
	// Most DNS servers have a timeout of two seconds, so we wait slightly longer
	dnsclient := dns.Client{Timeout: 2500 * time.Millisecond}
	message := dns.Msg{}
	message.SetQuestion(dns.Fqdn(s), dns.TypeA)
	response, _, err := dnsclient.Exchange(&message, dnsserver+":53")

	if err != nil {
		// Check the type of error we encountered
		if errors.Is(err, os.ErrDeadlineExceeded) {
			// If we run into a timeout, the DNSBL may be dead, so we assume not listed
			return false, nil
		} else {
			// For any other error, we assume listed
			return true, err
		}
	} else {
		return len(response.Answer) > 0, nil
	}
}

func is_listed (ip string, dnsbl string) (bool, error) {
	return has_dns_entry(reverse_ip(ip) + "." + dnsbl)
}
