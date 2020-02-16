package main

import "log"
import "time"
import "github.com/domodwyer/mailyak"
import "net/smtp"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"

// Global db handle
var db *sql.DB

func main() {
	var err error

        // Open database and check that table exists
        db, err = sql.Open("sqlite3", "./followup.db")
        if err != nil {
                log.Fatal(err)
        }
	check_schema()

	for {
		// Find next reminder, one by one
		id, recipient, subject, messageid := find_next_reminder()
		if len(recipient) > 0 {
			// Construct new mail object
			mail := mailyak.New(get_setting(`smtphost`)+":25", smtp.PlainAuth("", get_setting(`smtpuser`), get_setting(`smtppass`), get_setting(`smtphost`)))
//			mail := mailyak.New("localhost:25", nil)

			// Set recipient, subject and message-id to make sure it gets associated
			mail.To(recipient)
			mail.Subject(subject)
			mail.AddHeader(`In-Reply-To`, messageid)

			// Send mail and mark it as send
			if err := mail.Send(); err != nil {
				mark_as_done(id)
			}
		}

		// Wait a bit before next iteration
		time.Sleep(5 * time.Second)
	}
}

func check_schema() bool {
	stmt1, err1 := db.Prepare("CREATE TABLE IF NOT EXISTS reminders (id INTEGER PRIMARY KEY AUTOINCREMENT, sender TEXT, subject TEXT, messageid TEXT, timestamp BIGINT, status TEXT)")
	if err1 != nil {
		log.Fatal(err1)
	}

	_, err := stmt1.Exec()
	if err != nil {
		log.Fatal(err)
	}
	return true
}

func find_next_reminder() (int64, string, string, string) {
	epoch := time.Now().Unix()

	stmt1, err1 := db.Prepare("SELECT id, sender, subject, messageid FROM reminders WHERE timestamp <= ? AND status is null LIMIT 1")
	defer stmt1.Close()
	if err1 != nil {
		log.Fatal(err1)
		return -1, ``, ``, ``
	}

//	rows, err2 := stmt1.Query(epoch, ``)
	var id int64
	var sender string
	var subject string
	var messageid string
//	err2 := stmt1.QueryRow(epoch, ``).Scan(&id, &sender, &subject, &messageid)
	err2 := stmt1.QueryRow(epoch).Scan(&id, &sender, &subject, &messageid)
	if err2 != nil {
//		log.Fatal(err2)
		return -1, ``, ``, ``
	}
//	defer rows.Close()

	return id, sender, subject, messageid
}

func mark_as_done(id int64) bool {
	stmt1, _ := db.Prepare("UPDATE reminders SET status = 'SENT' WHERE id = ?")
	defer stmt1.Close()

	_, err := stmt1.Exec(id)
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
