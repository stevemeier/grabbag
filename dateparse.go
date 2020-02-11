package main

import "fmt"
import "regexp"
import "strconv"
import "strings"
import "time"
//import "github.com/davecgh/go-spew/spew"

const timezone = "CET"

func main() {
	var seconds int64
	seconds, _ = iso_to_seconds("1d@lordy.de")
	fmt.Printf("1d: %d\n", seconds)
	seconds, _ = iso_to_seconds("7d@lordy.de")
	fmt.Printf("7d: %d\n",seconds)
	seconds, _ = iso_to_seconds("14d@lordy.de")
	fmt.Printf("14d: %d\n", seconds)
	seconds, _ = iso_to_seconds("30d@lordy.de")
	fmt.Printf("30d: %d\n", seconds)
	seconds, _ = iso_to_seconds("0100@lordy.de")
	fmt.Printf("0100: %d\n", seconds)
	seconds, _ = iso_to_seconds("2000@lordy.de")
	fmt.Printf("2000: %d\n", seconds)

	fmt.Println("---")

	seconds, _ = iso_to_seconds("3am@lordy.de")
	fmt.Printf("3am: %d\n", seconds)
	seconds, _ = iso_to_seconds("9am@lordy.de")
	fmt.Printf("9am: %d\n", seconds)
	seconds, _ = iso_to_seconds("3pm@lordy.de")
	fmt.Printf("3pm: %d\n", seconds)
	seconds, _ = iso_to_seconds("9pm@lordy.de")
	fmt.Printf("9pm: %d\n", seconds)

	fmt.Println("---")

	seconds, _ = iso_to_seconds("mo@lordy.de")
	fmt.Printf("mo: %d\n", seconds)
	seconds, _ = iso_to_seconds("di@lordy.de")
	fmt.Printf("di: %d\n", seconds)
	seconds, _ = iso_to_seconds("mi@lordy.de")
	fmt.Printf("mi: %d\n", seconds)
	seconds, _ = iso_to_seconds("do@lordy.de")
	fmt.Printf("do: %d\n", seconds)
	seconds, _ = iso_to_seconds("fr@lordy.de")
	fmt.Printf("fr: %d\n", seconds)
	seconds, _ = iso_to_seconds("sa@lordy.de")
	fmt.Printf("sa: %d\n", seconds)
	seconds, _ = iso_to_seconds("so@lordy.de")
	fmt.Printf("so: %d\n", seconds)

	fmt.Println("---")

	seconds, _ = iso_to_seconds("jan1@lordy.de")
	fmt.Printf("jan1: %d\n", seconds)
	seconds, _ = iso_to_seconds("dez31@lordy.de")
	fmt.Printf("dez31: %d\n", seconds)
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
		day, _ = strconv.Atoi(re6data[2])
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
