package main

import "fmt"
import "log"
import "net/mail"
import "time"
import "os"
import "regexp"
import "strconv"
import "strings"
import "github.com/davecgh/go-spew/spew"
//import "github.com/DusanKasan/parsemail"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"

// Global db handle
var db *sql.DB

func main() {
	var err error

	db, err = sql.Open("sqlite3", "./followup.db")
	if err != nil {
		log.Fatal(err)
	}
	check_schema()

	var message *mail.Message
	message, err = mail.ReadMessage(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	if len(message.Header.Get("From")) == 0 {
		log.Fatal("No From Header!")
	}

	var from *mail.Address
	from, err = mail.ParseAddress(message.Header.Get("From"))

	fmt.Println("From: "+from.Address)

	var dest []string
	dest = append(dest, AddressesFromField(message.Header, "To")...)
	dest = append(dest, AddressesFromField(message.Header, "Cc")...)
	dest = append(dest, AddressesFromField(message.Header, "Bcc")...)
	spew.Dump(dest)

	for _, addr := range dest {
		duration, err := iso_to_seconds(addr)
//		fmt.Println(duration, err)
		if err == nil && duration > 0 {
			fmt.Println(addr, duration)
//			epoch := time.Now()
			sender, _ := mail.ParseAddress(message.Header.Get("From"))
			reminder_created := create_reminder(sender.Address, message.Header.Get("Subject"), message.Header.Get("Message-ID"), time.Now().Unix() + duration)
			fmt.Println(reminder_created)
		}
	}

//	email, err := parsemail.Parse(message.Body)
//	spew.Dump(email.Attachments)
//	spew.Dump(email)
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

	re1 := regexp.MustCompile(`(\d+)([m|h|d|w|m|y])`)
	re1data := re1.FindStringSubmatch(addrparts[0])
	if len(re1data) == 3 {
		if re1data[2] == "d" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 86400), nil
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
	_ = re3

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
	var day string
	if len(re5data) == 3 {
		month = re5data[1]
		day   = re5data[2]
	}
	if len(re6data) == 3 {
		day   = re6data[1]
		month = re6data[2]
	}
	if (day != "") && (month != "") {
	}

	return -1, nil
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

func ShortMonthToNumber(month string) int {
	mapping := map[string]int {
		"jan": 1,
		"feb": 2,
		"mar": 3, "mrz": 3,
		"apr": 4,
		"may": 5, "mai": 5,
		"jun": 6,
		"jul": 7,
		"aug": 8,
		"sep": 9,
		"oct": 10, "okt": 10,
		"nov": 11,
		"dec": 12, "dez": 12,
	}
	return mapping[strings.ToLower(month)]
}

func check_schema() bool {
	stmt1, err1 := db.Prepare("CREATE TABLE IF NOT EXISTS reminders (id INTEGER PRIMARY KEY AUTOINCREMENT, sender TEXT, subject TEXT, messageid TEXT, timestamp BIGINT)")
	if err1 != nil {
		log.Fatal(err1)
	}

	_, err := stmt1.Exec()
	if err != nil {
		log.Fatal(err)
	}
	return true
}
