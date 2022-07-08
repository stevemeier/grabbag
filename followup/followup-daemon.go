package main

import "crypto/tls"
import "fmt"
import "log"
import "time"
import "github.com/domodwyer/mailyak"
import "net/smtp"
import "os"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"
import "github.com/DavidGamba/go-getoptions"
import "github.com/davecgh/go-spew/spew"
import lib "./lib"

const version string = "20200305"

func main() {
	var db *sql.DB
	var err error
	var debug bool
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
	if lib.Get_setting(db,`smtphost`,``) == `` { log.Fatal(`ERROR: smtphost (server) not set`) }
	if lib.Get_setting(db,`smtpuser`,``) == `` { log.Fatal(`ERROR: smtpuser (username) not set`) }
	if lib.Get_setting(db,`smtppass`,``) == `` { log.Fatal(`ERROR: smtppass (password) not set`) }
	if lib.Get_setting(db,`smtpfrom`,``) == `` { log.Fatal(`ERROR: smtpfrom (sender) not set`) }

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
		id, recipient, subject, messageid, uuid, recurring, spec = find_next_reminder(db)
		if debug {
			spew.Dump(id, recipient, subject, messageid, uuid, spec)
		}
		if len(recipient) > 0 {
			if debug {
				fmt.Println("INFO: Found a pending reminder")
			}
			// Construct new mail object
			mail, newerr := mailyak.NewWithTLS(lib.Get_setting(db,`smtphost`,``)+":"+lib.Get_setting(db,`smtpport`,`25`),
						smtp.PlainAuth("",
					                   lib.Get_setting(db,`smtpuser`,``),
							   lib.Get_setting(db,`smtppass`,``),
							   lib.Get_setting(db,`smtphost`,``)),
							   &tls.Config{ServerName: lib.Get_setting(db,`smtphost`,``),
								       InsecureSkipVerify: len(lib.Get_setting(db,`smtpinsecure`,``)) > 0},
							   )

			if newerr != nil {
				log.Fatal(newerr)
			}

			// Set the sender
			mail.From(lib.Get_setting(db,`smtpfrom`,``))

			// Set recipient, subject and message-id to make sure it gets associated
			mail.To(recipient)
			mail.Subject(subject)
			mail.ReplyTo(uuid + `@` + lib.Domain_of(lib.Get_setting(db,`smtpfrom`,``)))
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
					success := mark_as_done(db, id)
					if debug {
						fmt.Printf("INFO: mark_as_done returned: %t\n", success)
					}
				} else {
					// Recurring reminders get updated
					success := update_recurring(db, id, spec, lib.Get_setting(db,`timezone`,`CET`))
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

func find_next_reminder(db *sql.DB) (int64, string, string, string, string, int, string) {
	epoch := time.Now().Unix()

	stmt1, err1 := db.Prepare("SELECT id, sender, subject, messageid, uuid, recurring, spec FROM reminders " +
	                          "WHERE timestamp <= ? AND (status IS null OR recurring > 0) LIMIT 1")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer stmt1.Close()

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

func mark_as_done(db *sql.DB, id int64) bool {
	stmt1, _ := db.Prepare("UPDATE reminders SET status = 'DONE@'||strftime('%s','now') WHERE id = ?")
	defer stmt1.Close()

	_, err := stmt1.Exec(id)
	return err == nil
}

func update_recurring(db *sql.DB, id int64, spec string, timezone string) bool {
	next, _, _ := lib.Parse_spec(spec, timezone)
	if next <= 0 {
		return false
	}

	stmt1, _ := db.Prepare("UPDATE reminders SET timestamp = timestamp + ?, status = 'SENT@'||strftime('%s','now') WHERE id = ?")
	defer stmt1.Close()

	_, err := stmt1.Exec(next, id)
	return err == nil
}
