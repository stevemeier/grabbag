package main

import (
	"log"
	"regexp"
	"strings"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/DavidGamba/go-getoptions"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-imap"
)

type Notification struct {
	id		uint32
	subject		string
	topic		string
	problem		bool
	recovery	bool
	dtstart		bool
	dtend		bool
}

type Pair struct {
	problem		uint32
	recovery	uint32
}

var c *client.Client
var err error

func main() {
	// Parse arguments
	var server string
	var username string
	var password string
	var sender string
	var debug bool
	opt := getoptions.New()
	opt.StringVar(&server, "server", "", opt.Required())
	opt.StringVar(&username, "username", "", opt.Required())
	opt.StringVar(&password, "password", "", opt.Required())
	opt.StringVar(&sender, "sender", "")
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

	// Connect to server
	c, _ = client.DialTLS(server+":993", nil)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		// Activate debug
		c.SetDebug(os.Stderr)
	}

	// Don't forget to logout
	defer c.Logout()

	// Login
	if err := c.Login(username, password); err != nil {
		log.Fatal(err)
	}

	// List mailboxes
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func () {
		done <- c.List("", "*", mailboxes)
	}()

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	// Select INBOX
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Fatal(err)
	}

	// Select all messages
	seqset := new(imap.SeqSet)
	seqset.AddRange(uint32(1), mbox.Messages)

	messages := make(chan *imap.Message, 10)
	done = make(chan error, 1)
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
	}()

	var data []Notification
        var problem_re = regexp.MustCompile(`^\[PROBLEM\] `)
        var recovery_re = regexp.MustCompile(`^\[RECOVERY\] `)
	var status_re = regexp.MustCompile(`\w+!$`)
        var dtstart_re = regexp.MustCompile(`^\[DOWNTIMESTART\] `)
        var dtend_re = regexp.MustCompile(`^\[DOWNTIMEEND\] `)
	for msg := range messages {
		// Check sender of email
		from := msg.Envelope.From[0].MailboxName+"@"+msg.Envelope.From[0].HostName
		if len(sender) > 0 && sender != from {
			if debug {
				log.Println("Skipping email from sender "+from)
			}
			continue
		}

		var this Notification
		this.id = msg.SeqNum
		this.subject = msg.Envelope.Subject
		log.Printf("Processing #%d -- %s\n", this.id, this.subject)

		this.problem = problem_re.MatchString(this.subject)
		if this.problem {
			this.topic = strings.Replace(this.subject, `[PROBLEM] `, ``, 1)
			this.topic = status_re.ReplaceAllString(this.topic, ``)
		}

		this.recovery = recovery_re.MatchString(this.subject)
		if this.recovery {
			this.topic = strings.Replace(this.subject, `[RECOVERY] `, ``, 1)
			this.topic = status_re.ReplaceAllString(this.topic, ``)
		}

		this.dtstart = dtstart_re.MatchString(this.subject)
		if this.dtstart {
			this.topic = strings.Replace(this.subject, `[DOWNTIMESTART] `, ``, 1)
			this.topic = status_re.ReplaceAllString(this.topic, ``)
		}

		this.dtend = dtend_re.MatchString(this.subject)
		if this.dtend {
			this.topic = strings.Replace(this.subject, `[DOWNTIMEEND] `, ``, 1)
			this.topic = status_re.ReplaceAllString(this.topic, ``)
		}

		data = append(data, this)
		spew.Dump(this)
	}

	var pairs []Pair
	var matchup = make(map[uint32]bool)
	for _, l1 := range data {
		for _, l2 := range data {
			if ((l1.problem && l2.recovery) || (l1.dtstart && l2.dtend)) &&
			   (l1.topic == l2.topic) &&
			   !matchup[l1.id] &&
			   !matchup[l2.id] &&
			   (l1.id < l2.id) {
				var this Pair
				this.problem = l1.id
				this.recovery = l2.id
				pairs = append(pairs, this)
				matchup[l1.id] = true
				matchup[l2.id] = true
				log.Printf("Deleting #%d\n", l1.id)
				DeleteMessage(l1.id)
				log.Printf("Deleting #%d\n", l2.id)
				DeleteMessage(l2.id)

				break
			}
		}
	}

	if err := <-done; err != nil {
		log.Fatal(err)
	}

	if len(pairs) > 0 {
		_ = c.Expunge(nil)
	}
}

func DeleteMessage(id uint32) bool {
	var err error
	delset := new(imap.SeqSet)
	delset.AddRange(id, id)
	err = c.Store(delset, "+FLAGS", []interface{}{imap.DeletedFlag}, nil)
	return err == nil
}
