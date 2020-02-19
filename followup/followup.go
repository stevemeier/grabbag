package main

import "errors"
import "fmt"
import "log"
import "net/mail"
import "os"
import "regexp"
import "strconv"
import "strings"
import "time"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"
import "github.com/DavidGamba/go-getoptions"

// Local timezone
const timezone = "CET"

// Global db handle
var db *sql.DB

func main() {
	var err error
	var debug bool

	// Parse options
	opt := getoptions.New()
	opt.BoolVar(&debug, "debug", false)
	_, _ = opt.Parse(os.Args[1:])

	// Open database and check that table exists
	var dbpath string
	if env_defined("HOME") {
		dbpath = os.Getenv("HOME") + "/followup.db"
	} else {
		dbpath = "./followup.db"
	}
	if debug { fmt.Println("DB is in "+dbpath) }
	db, err = sql.Open("sqlite3", dbpath)
	if err != nil {
		log.Fatal(err)
	}
	check_schema()

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

	// Go through all addresses
	for _, addr := range dest {
		// Change address into seconds in the future
		duration, err := iso_to_seconds(addr)
		if debug {
			fmt.Println(addr, duration)
		}
		if err == nil && duration > 0 {
			// Create a reminder to be send later
			reminder_created := create_reminder(from.Address, message.Header.Get("Subject"), message.Header.Get("Message-ID"), time.Now().Unix() + duration)
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

func create_reminder (from string, subject string, messageid string, when int64) bool {
	_, err := db.Exec(`INSERT INTO reminders (sender, subject, messageid, timestamp) VALUES ("` + from + `","` + subject + `","` + messageid + `","` + strconv.FormatInt(when, 10) + `")`)
	if err != nil {
		log.Fatal(err)
	}
	return err == nil
}

func iso_to_seconds (address string) (int64, error) {
        addrparts := strings.Split(address, "@")

	re1 := regexp.MustCompile(`(\d+)([h|d|w|m|y])$`)
	re1data := re1.FindStringSubmatch(addrparts[0])
	if len(re1data) == 3 {
		if re1data[2] == "h" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 3600), nil
		}
		if re1data[2] == "d" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 86400), nil
		}
		if re1data[2] == "w" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 604800), nil
		}
		if re1data[2] == "m" {
			// Unlike hour, day and week, month has no fixed number of seconds
			count, _ := strconv.Atoi(re1data[1])
			goal := time.Now().AddDate(0,count,0)
			return int64(goal.Sub(time.Now()).Seconds()), nil
		}
		if re1data[2] == "y" {
			// Unlike hour, day and week, year has no fixed number of seconds
			count, _ := strconv.Atoi(re1data[1])
			goal := time.Now().AddDate(count,0,0)
			return int64(goal.Sub(time.Now()).Seconds()), nil
		}
	}

	re2 := regexp.MustCompile(`(\d{1,2})(\d{2})`)
	re2data := re2.FindStringSubmatch(addrparts[0])
	if len(re2data) == 3 {
		hour, _ := strconv.Atoi(re2data[1])
		minute, _ := strconv.Atoi(re2data[2])
		goalsecond := hour * 3600 + minute * 60
		if goalsecond > getSecondOfDay(time.Now()) {
			return int64(goalsecond - getSecondOfDay(time.Now())), nil
		} else {
			return int64(86400 - getSecondOfDay(time.Now()) + goalsecond), nil
		}
	}

	re3 := regexp.MustCompile(`(\d{1,2})(am|pm)`)
	re3data := re3.FindStringSubmatch(addrparts[0])
	if len(re3data) == 3 {
		hour, _ := strconv.Atoi(re3data[1])
		if (re3data[2] == "pm") {
			hour += 12
		}
		if (hour * 3600) > getSecondOfDay(time.Now()) {
			// in the future
			return int64((hour * 3600) - getSecondOfDay(time.Now())), nil
		} else {
			// in the past
			return int64(86400 - (getSecondOfDay(time.Now()) - (hour * 3600))), nil
		}

	}

	re4 := regexp.MustCompile(`^(mo|tu|di|we|mi|th|do|fr|sa|su|so)`)
	re4data := re4.FindStringSubmatch(addrparts[0])
	if len(re4data) == 2 {
		if ShortDayToNumber(re4data[1]) > int(time.Now().Weekday()) {
			return int64((ShortDayToNumber(re4data[1]) - int(time.Now().Weekday())) * 86400), nil
		}
		if ShortDayToNumber(re4data[1]) == int(time.Now().Weekday()) {
			return 604800, nil
		}
		if ShortDayToNumber(re4data[1]) < int(time.Now().Weekday()) {
			return int64(604800 - (int(time.Now().Weekday()) - ShortDayToNumber(re4data[1])) * 86400), nil
		}
	}

	re5 := regexp.MustCompile(`^(jan|feb|mar|mrz|apr|may|mai|jun|jul|aug|sep|oct|okt|nov|dec|dez)[a-z]{0,}(\d+)`)
	re5data := re5.FindStringSubmatch(addrparts[0])
	re6 := regexp.MustCompile(`^(\d+)(jan|feb|mar|mrz|apr|may|mai|jun|jul|aug|sep|oct|okt|nov|dec|dez)`)
	re6data := re6.FindStringSubmatch(addrparts[0])
	var month string
	var day int
	if len(re5data) == 3 {
		month  = re5data[1]
		day, _ = strconv.Atoi(re5data[2])
	}
	if len(re6data) == 3 {
		day, _ = strconv.Atoi(re6data[1])
		month  = re6data[2]
	}
	if (day > 0) && (month != "") {
		location, _ := time.LoadLocation(timezone)
		goal := time.Date(time.Now().Year(), ShortMonthToNumber(month), day, 0, 0, 0, 0, location)
		if goal.Sub(time.Now()).Seconds() < 0 {
			goal = time.Date(time.Now().Year()+1, ShortMonthToNumber(month), day, 0, 0, 0, 0, location)
		}
		return int64(goal.Sub(time.Now()).Seconds()), nil
	}

	return -1, errors.New("Could not parse this: "+addrparts[0])
}

func getSecondOfDay(t time.Time) int {
	// https://stackoverflow.com/questions/55023060/how-to-get-the-seconds-of-day
	return 60*60*t.Hour() + 60*t.Minute() + t.Second()
}

func ShortDayToNumber(day string) int {
	mapping := map[string]int {
		"su": 0, "so": 0,
		"mo": 1,
		"tu": 2, "di": 2,
		"we": 3, "mi": 3,
		"th": 4, "do": 4,
		"fr": 5,
		"sa": 6,
	}
	return mapping[strings.ToLower(day)]
}

func ShortMonthToNumber(month string) time.Month {
	mapping := map[string]time.Month {
		"jan": time.January,
		"feb": time.February,
		"mar": time.March, "mrz": time.March,
		"apr": time.April,
		"may": time.May, "mai": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October, "okt": time.October,
		"nov": time.November,
		"dec": time.December, "dez": time.December,
	}
	return mapping[strings.ToLower(month)]
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

func env_defined(key string) bool {
        _, exists := os.LookupEnv(key)
        return exists
}