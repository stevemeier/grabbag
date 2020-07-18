package main

import "github.com/miekg/dns"
import "github.com/DavidGamba/go-getoptions"
import "log"
import "net"
import "os"
import "regexp"
import "strconv"
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

type ResourcePool struct {
	Client	*dns.Client
	Conn	*dns.Conn
	Uses	int64
}

// These are global for easy re-use
var resolver string
var resport int
var timeout int

func main() {
	// Parse options
        opt := getoptions.New()
	var dbfile string
	var workers int
	opt.StringVar(&dbfile, "db", "in-addr.sql", opt.Required())
	opt.StringVar(&resolver, "resolver", "193.189.250.100")
	opt.IntVar(&resport, "port", 53)
	opt.IntVar(&workers, "workers", 10)
	opt.IntVar(&timeout, "timeout", 6)
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
	ipqueue := make(chan string, workers * 100)

	// PTR Queue as channel (results)
	ptrqueue := make(chan Result, workers * 100)

	// Prepare resource pool
	respool := make(chan ResourcePool, workers)

	for i := 1; i <= cap(respool); i++ {
		c := init_dns_client(timeout)
		conn := init_dns_conn(c, resolver, resport)
//		respool <- ResourcePool{c, conn}
		respool <- ResourcePool{c, conn, 0}
	}

	// Statistics channel
	statchan := make(chan int, 1000)

	go find_next_ip(db, ipqueue)
	go store_results_tx(db, ptrqueue, statchan)
	go stat_printer(statchan)

	workqueue := make(chan bool, workers)

	log.Println("Entering for loop")
	for {
		// Add to the queue
		workqueue <- true
		go worker(workqueue, ipqueue, ptrqueue, respool)
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

func ptrlookup (ipaddr string, client *dns.Client, conn *dns.Conn) (*dns.Msg, error) {
	m1 := new(dns.Msg)
	m1.Id = dns.Id()
	m1.RecursionDesired = true
	m1.Question = make([]dns.Question, 1)
	m1.Question[0] = dns.Question{ReverseIPAddress(ipaddr) + ".in-addr.arpa.", dns.TypePTR, dns.ClassINET}

	in, _, err := client.ExchangeWithConn(m1, conn)

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
		if len(ipqueue) < (cap(ipqueue) / 2) {
		//	stmt1, err1 := db.Prepare("SELECT o1 || '.' || o2 || '.' || o3 || '.' || o4 FROM t1 WHERE lastupd IS NULL LIMIT ?")
			stmt1, err1 := db.Prepare("SELECT o1 || '.' || o2 || '.' || o3 || '.' || o4 FROM t1 ORDER BY lastupd ASC LIMIT ?")
//			stmt1, err1 := db.Prepare("SELECT o1 || '.' || o2 || '.' || o3 || '.' || o4 FROM t1 WHERE lastupd = 0 LIMIT ?")
			defer stmt1.Close()
			if err1 != nil {
				log.Fatal(err1)
			}

			var nextip string
			rows, selecterr := stmt1.Query(int(cap(ipqueue) / 2))
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
		time.Sleep(1 * time.Second)
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

func store_results_tx (db *sql.DB, ptrqueue chan Result, statchan chan int) {
	for {
		queuesize := len(ptrqueue)
		// If queue is 50% full, write to DB
		if queuesize > (cap(ptrqueue) / 2) {
			log.Printf("PTRqueue status: %d/%d\n", queuesize, cap(ptrqueue))

			// Open a transaction
			tx, _ := db.Begin()

			for i := 1; i < queuesize; i++ {
				result := <-ptrqueue
				oct := strings.Split(result.Ip, ".")
				stmt1, _ := tx.Prepare("UPDATE t1 SET rcode = ?, ptr = ?, lastupd = ? WHERE o1 = ? AND o2 = ? AND o3 = ? AND o4 = ?")
				defer stmt1.Close()

				now := time.Now()
				_, err := stmt1.Exec(result.Opcode, result.Ptrdata, now.Unix(), oct[0], oct[1], oct[2], oct[3])
				if err != nil {
					log.Println(err)
				}
			}
			// Commit transaction
			tx.Commit()

			// Write to stats channel
			statchan <- queuesize
		}
		time.Sleep(1 * time.Second)
	}
}

func worker (workqueue chan bool, ipqueue chan string, ptrqueue chan Result, respool chan ResourcePool) {
	nextip := <-ipqueue

	myresource :=  <-respool
	c := myresource.Client
	conn := myresource.Conn
	myresource.Uses++

	in, lookuperr := ptrlookup(nextip, c, conn)
	if lookuperr == nil {
		ptrqueue <- Result{Ip: nextip, Opcode: opcode(in), Ptrdata: ptrdata(in)}
//		log.Printf("%s: %d, %s\n", nextip, opcode(in), ptrdata(in))
	} else {
		log.Printf("%s: %s\n", nextip, lookuperr.Error())
	}

	// Return resources
	if lookuperr == nil {
		respool <- myresource
	} else {
		// If we encountered an error, we do not recycle the connection
		newclient := init_dns_client(timeout)
		respool <- ResourcePool{newclient, init_dns_conn(newclient, resolver, resport), 0}
	}

	// Free a spot in workqueue
	_ = <-workqueue
}

func stat_printer (statchan chan int) {
	var interval int = 60
	for {
		entries := len(statchan)
		var total int

		for i := 1; i < entries; i++ {
			total += <-statchan
		}

		log.Printf("Database commits: %d per second\n", int(total / interval))
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func init_dns_client (timeout int) (*dns.Client) {
	c := new(dns.Client)
	c.Timeout = time.Duration(timeout) * time.Second
	c.ReadTimeout = time.Duration(timeout) * time.Second
	c.WriteTimeout = time.Duration(timeout) * time.Second
	return c
}

func init_dns_conn (c *dns.Client, resolver string, resport int) (*dns.Conn) {
	conn, _ := c.Dial(resolver+":"+strconv.FormatInt(int64(resport), 10))
	return conn
}
