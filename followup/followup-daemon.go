package main

import "fmt"
import "log"
import "time"
import "github.com/domodwyer/mailyak"
import "net/smtp"
import "os"
import "strconv"
import "strings"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"
import "github.com/DavidGamba/go-getoptions"
import "github.com/davecgh/go-spew/spew"
import lib "./lib"

const version string = "20200305"
const timezone = "CET" // move to SQLite settings

// Global db handle
var db *sql.DB
var debug bool

func main() {
	var err error
	var dbpath string

	// Parse options
	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	opt.StringVar(&dbpath, "db", "")
	_, _ = opt.Parse(os.Args[1:])

        // Open database and check that table exists
	if lib.Env_defined("DBPATH") { dbpath = os.Getenv("DBPATH") }
        if (dbpath == ``) {
	        if lib.Env_defined("HOME") {
			dbpath = os.Getenv("HOME") + "/followup.db"
	        } else {
			dbpath = "./followup.db"
		}
        }

        if debug { fmt.Println("DB is in "+dbpath) }
        db, err = sql.Open("sqlite3", dbpath)
        if err != nil {
                log.Fatal(err)
        }
	lib.Check_schema(db)

	// Check settings
	if get_setting(`smtphost`) == `` { log.Fatal(`ERROR: smtphost (server) not set`) }
	if get_setting(`smtpuser`) == `` { log.Fatal(`ERROR: smtpuser (username) not set`) }
	if get_setting(`smtppass`) == `` { log.Fatal(`ERROR: smtppass (password) not set`) }
	if get_setting(`smtpfrom`) == `` { log.Fatal(`ERROR: smtpfrom (sender) not set`) }

	for {
		// Find next reminder, one by one
		var id int64
		var recipient string
		var subject string
		var messageid string
		var uuid string
		var recurring int
		var spec string
		if debug {
			fmt.Println("INFO: Scanning for reminders")
		}
		id, recipient, subject, messageid, uuid, recurring, spec = find_next_reminder()
		if debug {
			spew.Dump(id, recipient, subject, messageid, uuid, spec)
		}
		if len(recipient) > 0 {
			if debug {
				fmt.Println("INFO: Found a pending reminder")
			}
			// Construct new mail object
			mail := mailyak.New(get_setting(`smtphost`)+":25",
			                    smtp.PlainAuth("", get_setting(`smtpuser`), get_setting(`smtppass`), get_setting(`smtphost`)))
			mail.From(get_setting(`smtpfrom`))

			// Set recipient, subject and message-id to make sure it gets associated
			mail.To(recipient)
			mail.Subject(subject)
			mail.ReplyTo(uuid + `@` + domain_of(get_setting(`smtpfrom`)))
			mail.AddHeader(`In-Reply-To`, messageid)
			mail.AddHeader(`X-Followup-Version`, version)

			// Recurring
			if recurring > 0 {
				mail.Plain().Set("This is a recurring reminder. Reply to cancel.")
			} else {
				mail.Plain().Set("This is a one-time reminder.")
			}

			if debug {
				fmt.Println("INFO: Sending reminder to "+recipient)
			}

			// Send mail and mark it as send
			err := mail.Send()
			if err == nil {
				if recurring == 0 {
					// One-time reminders get marked done
					success := mark_as_done(id)
					if debug {
						fmt.Printf("INFO: mark_as_done returned: %t\n", success)
					}
				} else {
					// Recurring reminders get updated
					success := update_recurring(id, spec)
					if debug {
						fmt.Printf("INFO: update_recurring returned: %t\n", success)
					}
				}
			} else {
				log.Fatal(err)
			}
		}

		// Wait a bit before next iteration
		time.Sleep(5 * time.Second)
	}
}

func find_next_reminder() (int64, string, string, string, string, int, string) {
	epoch := time.Now().Unix()

	stmt1, err1 := db.Prepare("SELECT id, sender, subject, messageid, uuid, recurring, spec FROM reminders WHERE timestamp <= ? AND status IS null LIMIT 1")
	defer stmt1.Close()
	if err1 != nil {
		log.Fatal(err1)
	}

	var id int64
	var sender string
	var subject string
	var messageid string
	var uuid string
	var recurring int
	var spec string

	err2 := stmt1.QueryRow(epoch).Scan(&id, &sender, &subject, &messageid, &uuid, &recurring, &spec)
	if err2 == sql.ErrNoRows {
		// No data in the database
		return -1, ``, ``, ``, ``, -1, ``
	}
	if err2 != nil {
		// Unknown error
		log.Fatal(err2)
	}
	defer stmt1.Close()

	// Return found data
	return id, sender, subject, messageid, uuid, recurring, spec
}

func mark_as_done(id int64) bool {
	stmt1, _ := db.Prepare("UPDATE reminders SET status = ? WHERE id = ?")
	defer stmt1.Close()

	_, err := stmt1.Exec(`SENT@` + strconv.FormatInt(time.Now().Unix(), 10), id)
	return err == nil
}

func update_recurring(id int64, spec string) bool {
	next, _, _ := lib.Parse_spec(spec, timezone)
	if next <= 0 {
		return false
	}

	stmt1, _ := db.Prepare("UPDATE reminders SET timestamp = timestamp + ? WHERE id = ?")
	defer stmt1.Close()

	_, err := stmt1.Exec(next, id)
	return err == nil
}

func get_setting(name string) string {
	var result string

	stmt1, err1 := db.Prepare("SELECT value FROM settings where name = ? LIMIT 1")
	defer stmt1.Close()
	if err1 != nil {
		return ``
	}

	err2 := stmt1.QueryRow(name).Scan(&result)
	if err2 != nil {
		return ``
	} else {
		return result
	}
}

func domain_of (address string) string {
	addrparts := strings.Split(address, "@")
	if len(addrparts) == 2 {
		return addrparts[1]
	} else {
		return address
	}
}
