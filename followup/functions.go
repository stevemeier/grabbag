package main

import "errors"
import "log"
import "regexp"
import "os"
import "strconv"
import "strings"
import "time"
import "database/sql"
import _ "github.com/mattn/go-sqlite3"

func Env_defined(key string) bool {
        _, exists := os.LookupEnv(key)
        return exists
}

func Check_schema(db *sql.DB) bool {
	var err error
	stmt1, err1 := db.Prepare("CREATE TABLE IF NOT EXISTS reminders (" +
	                          "id INTEGER PRIMARY KEY AUTOINCREMENT," +
				  "uuid TEXT," +
				  "sender TEXT," +
				  "subject TEXT," +
				  "messageid TEXT," +
				  "timestamp BIGINT," +
				  "recurring INTEGER," +
				  "spec TEXT," +
				  "status TEXT)")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer stmt1.Close()

	_, err = stmt1.Exec()
	if err != nil {
		log.Fatal(err)
	}

	stmt2, err2 := db.Prepare("CREATE TABLE IF NOT EXISTS settings (name PRIMARY KEY NOT NULL, value TEXT)")
	if err2 != nil {
		log.Fatal(err2)
	}
	defer stmt2.Close()

	_, err = stmt2.Exec()
	if err != nil {
		log.Fatal(err)
	}

	return true
}

func Is_uuid(input string) bool {
	// Regex from:
	// https://github.com/ramsey/uuid/blob/c141cdc8dafa3e506f69753b692f6662b46aa933/src/Uuid.php#L96
	match, _ := regexp.MatchString("^[0-9A-Fa-f]{8}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{4}-[0-9A-Fa-f]{12}$", input)
	return match
}

func Disable_reminder(db *sql.DB, addr string) bool {
	stmt1, err1 := db.Prepare("UPDATE reminders SET recurring = 0, status = 'DISABLED@'||strftime('%s','now') WHERE uuid = ?")
	if err1 != nil {
		log.Fatal(err1)
	}
	defer stmt1.Close()

	_, err := stmt1.Exec(addr)
	return err == nil
}

func User_of (address string) string {
	addrparts := strings.Split(address, "@")
	if len(addrparts) == 2 {
		return addrparts[0]
	} else {
		return address
	}
}

func Domain_of (address string) string {
        addrparts := strings.Split(address, "@")
        if len(addrparts) == 2 {
                return addrparts[1]
        } else {
                return address
        }
}

func Parse_spec (address string, timezone string) (int64, int, error) {
	// Recurring support
	var recurring int = 0
	plusre := regexp.MustCompile(`\+$`)
	if plusre.MatchString(address) {
		recurring = 1
	}

	re1 := regexp.MustCompile(`^(\d+)([h|d|w|m|y])`)
	re1data := re1.FindStringSubmatch(address)
	if len(re1data) == 3 {
		if re1data[2] == "h" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 3600), recurring, nil
		}
		if re1data[2] == "d" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 86400), recurring, nil
		}
		if re1data[2] == "w" {
			count, _ := strconv.Atoi(re1data[1])
			return int64(count * 604800), recurring, nil
		}
		if re1data[2] == "m" {
			// Unlike hour, day and week, month has no fixed number of seconds
			count, _ := strconv.Atoi(re1data[1])
			goal := time.Now().AddDate(0,count,0)
			return int64(goal.Sub(time.Now()).Seconds()), recurring, nil
		}
		if re1data[2] == "y" {
			// Unlike hour, day and week, year has no fixed number of seconds
			count, _ := strconv.Atoi(re1data[1])
			goal := time.Now().AddDate(count,0,0)
			return int64(goal.Sub(time.Now()).Seconds()), recurring, nil
		}
	}

	re2 := regexp.MustCompile(`^(\d{1,2})(\d{2})`)
	re2data := re2.FindStringSubmatch(address)
	if len(re2data) == 3 {
		hour, _ := strconv.Atoi(re2data[1])
		minute, _ := strconv.Atoi(re2data[2])
		goalsecond := hour * 3600 + minute * 60
		if goalsecond > getSecondOfDay(time.Now()) {
			return int64(goalsecond - getSecondOfDay(time.Now())), recurring, nil
		} else {
			return int64(86400 - getSecondOfDay(time.Now()) + goalsecond), recurring, nil
		}
	}

	re3 := regexp.MustCompile(`^(\d{1,2})(am|pm)`)
	re3data := re3.FindStringSubmatch(address)
	if len(re3data) == 3 {
		hour, _ := strconv.Atoi(re3data[1])
		if (re3data[2] == "pm") {
			hour += 12
		}
		if (hour * 3600) > getSecondOfDay(time.Now()) {
			// in the future
			return int64((hour * 3600) - getSecondOfDay(time.Now())), recurring, nil
		} else {
			// in the past
			return int64(86400 - (getSecondOfDay(time.Now()) - (hour * 3600))), recurring, nil
		}

	}

	re4 := regexp.MustCompile(`^(mo|tu|di|we|mi|th|do|fr|sa|su|so)`)
	re4data := re4.FindStringSubmatch(address)
	if len(re4data) == 2 {
		if ShortDayToNumber(re4data[1]) > int(time.Now().Weekday()) {
			return int64((ShortDayToNumber(re4data[1]) - int(time.Now().Weekday())) * 86400), recurring, nil
		}
		if ShortDayToNumber(re4data[1]) == int(time.Now().Weekday()) {
			return 604800, recurring, nil
		}
		if ShortDayToNumber(re4data[1]) < int(time.Now().Weekday()) {
			return int64(604800 - (int(time.Now().Weekday()) - ShortDayToNumber(re4data[1])) * 86400), recurring, nil
		}
	}

	re5 := regexp.MustCompile(`^(jan|feb|mar|mrz|apr|may|mai|jun|jul|aug|sep|oct|okt|nov|dec|dez)[a-z]{0,}(\d+)`)
	re5data := re5.FindStringSubmatch(address)
	re6 := regexp.MustCompile(`^(\d+)(jan|feb|mar|mrz|apr|may|mai|jun|jul|aug|sep|oct|okt|nov|dec|dez)`)
	re6data := re6.FindStringSubmatch(address)
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
		return int64(goal.Sub(time.Now()).Seconds()), recurring, nil
	}

	return -1, recurring, errors.New("Could not parse this: "+address)
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

func Get_setting(db *sql.DB, name string, undef string) string {
	var result string

	stmt1, err1 := db.Prepare("SELECT value FROM settings where name = ? LIMIT 1")
	defer stmt1.Close()
	if err1 != nil {
		return undef
	}

	err2 := stmt1.QueryRow(name).Scan(&result)
	if err2 != nil {
		return undef
	} else {
		return result
	}
}
