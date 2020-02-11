package main

import "fmt"
import "regexp"
import "strconv"
import "strings"
import "time"
//import "github.com/davecgh/go-spew/spew"

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

//	fmt.Println(int(time.Now().Weekday()))
//	fmt.Println(int(time.Now().Month()))
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
