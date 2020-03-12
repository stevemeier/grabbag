package main

import "fmt"
import "log"
import "net/mail"
import "os"
import "strconv"
import "time"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"
import "github.com/DavidGamba/go-getoptions"
import "github.com/gofrs/uuid"
import lib "./lib"

func main() {
	var db *sql.DB
	var err error
	var debug bool

	// Set up a function to catch panic and exit with default code
        defer func() {
                if err := recover(); err != nil {
                        log.Println(err)
                        os.Exit(111)
                }
        }()

	// Parse options
	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	_, _ = opt.Parse(os.Args[1:])

	// Open database and check that table exists
	var dbpath string
	if lib.Env_defined("HOME") {
		dbpath = os.Getenv("HOME") + "/followup.db"
	} else {
		dbpath = "./followup.db"
	}
	if debug { fmt.Println("DB is in "+dbpath) }
	db, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatal(err)
	}
	lib.Check_schema(db)

	// Read eEmail from STDIN
	var message *mail.Message
	message, err = mail.ReadMessage(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	// Check that there is a From: address we can reply to
	if len(message.Header.Get("From")) == 0 {
		log.Fatal("No From Header!")
	}

	// Parse the sender address
	var from *mail.Address
	from, err = mail.ParseAddress(message.Header.Get("From"))

	// Extract To, CC and Bcc fields for processing
	var dest []string
	dest = append(dest, AddressesFromField(message.Header, "To")...)
	dest = append(dest, AddressesFromField(message.Header, "Cc")...)
	dest = append(dest, AddressesFromField(message.Header, "Bcc")...)

	// Process $RECIPIENT from environment, if set
	if lib.Env_defined("ORIGINAL_RECIPIENT") {
		dest = append(dest, os.Getenv("ORIGINAL_RECIPIENT"))
	}

	// Go through all addresses
	for _, addr := range dest {
		if debug {
			fmt.Printf("Processing %s\n", addr)
		}
		// Remove recurring reminders when replied to
		if lib.Is_uuid(lib.User_of(addr)) {
			if debug {
				fmt.Printf("Disabling reminder for %s\n", lib.User_of(addr))
			}
			if lib.Disable_reminder(db, lib.User_of(addr)) {
				os.Exit(0)
			} else {
				os.Exit(111)
			}
		}

		// Change address into seconds in the future
		duration, recurring, err := lib.Parse_spec(lib.User_of(addr), lib.Get_setting(db,`timezone`,`CET`))
		if debug {
			fmt.Println(addr, duration)
		}
		if err == nil && duration > 0 {
			// Create a reminder to be send later
			reminder_created := create_reminder(db,
			                                    from.Address,
			                                    message.Header.Get("Subject"),
							    message.Header.Get("Message-ID"),
							    time.Now().Unix() + duration,
						            recurring,
							    lib.User_of(addr) )
			if reminder_created {
				os.Exit(0)
			} else {
				os.Exit(111)
			}
		}
	}
}

func AddressesFromField (header mail.Header, field string) ([]string) {
	var result []string

	addresslist, _ := mail.ParseAddressList(header.Get(field))
	for _, obj := range addresslist {
		result = append(result, obj.Address)
	}

	return result
}

func create_reminder (db *sql.DB, from string, subject string, messageid string, when int64, recurring int, spec string) bool {
	uuid, err1 := uuid.NewV4()
	if err1 != nil {
		log.Fatal(err1)
	}
	_, err2 := db.Exec(`INSERT INTO reminders (uuid, sender, subject, messageid, timestamp, recurring, spec) VALUES ("` +
                                                   uuid.String() + `","` +
						   from + `","` +
						   subject + `","` +
						   messageid + `","` +
						   strconv.FormatInt(when, 10) + `","` +
						   strconv.FormatInt(int64(recurring), 10) + `","` +
						   spec +
						   `")`)
	if err2 != nil {
		log.Fatal(err2)
	}
	return err2 == nil
}
