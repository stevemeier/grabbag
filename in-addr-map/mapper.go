package main

import "github.com/miekg/dns"
import "github.com/DavidGamba/go-getoptions"
import "log"
import "net"
import "os"
import "regexp"
import "strings"
import "time"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"

// 8.8.8.8.in-addr.arpa. => dns.google.
// 4.3.2.1.in-addr.arpa. => NXDOMAIN (in.Answer is empty)
// 1.1.145.199.in-addr.arpa => SERVFAIL (probably DNSSEC) (Authority Section is empty)
// 1.1.26.166.in-addr.arpa => SERVFAIL (everywhere, not DNSSEC)

type Result struct {
	Ip	string
	Opcode	int
	Ptrdata	string
}

func main() {
	// Parse options
        opt := getoptions.New()
	var dbfile string
	var resolver string
	var workers int
	opt.StringVar(&dbfile, "db", "in-addr.sql", opt.Required())
	opt.StringVar(&resolver, "resolver", "193.189.250.100")
	opt.IntVar(&workers, "workers", 10)
        remaining, err := opt.Parse(os.Args[1:])
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

	// Connect to database
	var db *sql.DB
	var dberr error
	db, dberr = sql.Open("sqlite3", dbfile)
        if dberr != nil {
                log.Fatal(dberr)
        }
	log.Println("Connected to DB")
	defer db.Close()

	// IP Queue as channel
	ipqueue := make(chan string, 100)

	// PTR Queue as channel (results)
	ptrqueue := make(chan Result, 100)

	c := new(dns.Client)
	c.Timeout = 5 * time.Second

	go find_next_ip(db, ipqueue)
	go store_results(db, ptrqueue)

	workqueue := make(chan bool, workers)

	log.Println("Entering for loop")
	for {
		// Add to the queue
		workqueue <- true
		go worker(workqueue, ipqueue, ptrqueue, c, resolver)
	}

} // end of main

func opcode (msg *dns.Msg) (int) {
	re1 := regexp.MustCompile(`status: (.*?),`)
	rcode := re1.FindStringSubmatch(msg.String())
	if len(rcode) == 2 {
		if rcode[1] == `NOERROR`  { return 0 }
		if rcode[1] == `SERVFAIL` { return 2 }
		if rcode[1] == `NXDOMAIN` { return 3 }
		if rcode[1] == `REFUSED`  { return 5 }
	}
	return 65535
}

func ptrdata (msg *dns.Msg) (string) {
	var data []string
	re1 := regexp.MustCompile(`PTR\s+(.*)$`)

	if len(msg.Answer) == 0 { return ``}

	for i, _ := range(msg.Answer) {
		if t, ok := msg.Answer[i].(*dns.PTR); ok {
			ptr := re1.FindStringSubmatch(t.String())
			if len(ptr) == 2 {
				data = append(data, ptr[1])
			}
		}
	}

	return strings.Join(data, `/`)
}

func ptrlookup (ipaddr string, c *dns.Client, conn *dns.Conn) (*dns.Msg, error) {
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{ReverseIPAddress(ipaddr) + ".in-addr.arpa.", dns.TypePTR, dns.ClassINET}

	in, _, err := c.ExchangeWithConn(m1, conn)

	return in, err
}

func ReverseIPAddress (input string) (string) {
	ip := net.ParseIP(input)

	// Source: https://socketloop.com/tutorials/golang-reverse-ip-address-for-reverse-dns-lookup-example
	if ip.To4() != nil {
		// split into slice by dot .
		addressSlice := strings.Split(ip.String(), ".")
		reverseSlice := []string{}

		for i := range addressSlice {
		     octet := addressSlice[len(addressSlice)-1-i]
		     reverseSlice = append(reverseSlice, octet)
		}

		return strings.Join(reverseSlice, ".")
	} else {
		log.Fatal("invalid IPv4 address: " + input)
		return ``
	}
}

func find_next_ip (db *sql.DB, ipqueue chan string) () {
	for {
		if len(ipqueue) == 0 {
			stmt1, err1 := db.Prepare("SELECT o1 || '.' || o2 || '.' || o3 || '.' || o4 FROM t1 WHERE lastupd IS NULL LIMIT ?")
			defer stmt1.Close()
			if err1 != nil {
				log.Fatal(err1)
			}

			var nextip string
			rows, selecterr := stmt1.Query(cap(ipqueue))
			if selecterr != nil {
				log.Println(selecterr)
			}
			defer rows.Close()

			for rows.Next() {
				rows.Scan(&nextip)
				ipqueue <- nextip
			}
			log.Printf("IPqueue status: %d/%d\n", len(ipqueue), cap(ipqueue))
		}
	}
}


func store_results (db *sql.DB, ptrqueue chan Result) {
	for {
		result := <-ptrqueue
		oct := strings.Split(result.Ip, ".")
		stmt1, _ := db.Prepare("UPDATE t1 SET rcode = ?, ptr = ?, lastupd = ? WHERE o1 = ? AND o2 = ? AND o3 = ? AND o4 = ?")
		defer stmt1.Close()

		now := time.Now()
		_, err := stmt1.Exec(result.Opcode, result.Ptrdata, now.Unix(), oct[0], oct[1], oct[2], oct[3])
		if err != nil {
			log.Println(err)
		}
//		log.Printf("PTRqueue status: %d/%d\n", len(ptrqueue), cap(ptrqueue))
	}
}

func worker (workqueue chan bool, ipqueue chan string, ptrqueue chan Result, c *dns.Client, resolver string) {
	nextip := <-ipqueue

	conn, connerr := c.Dial(resolver+":53")
	if connerr != nil {
		log.Println(connerr)
	}
	defer conn.Close()

	in, lookuperr := ptrlookup(nextip, c, conn)
	if lookuperr == nil {
		ptrqueue <- Result{Ip: nextip, Opcode: opcode(in), Ptrdata: ptrdata(in)}
		log.Printf("%s: %d, %s\n", nextip, opcode(in), ptrdata(in))
	} else {
		log.Printf("%s: %s\n", nextip, lookuperr.Error())
	}

	// Free a spot in workqueue
	_ = <-workqueue
}
