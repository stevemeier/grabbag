package main

import "errors"
import "fmt"
import "os"
import "regexp"
import "strconv"
import "strings"
import "github.com/DavidGamba/go-getoptions"
import "github.com/mackerelio/go-osstat/uptime"
import "github.com/olorin/nagiosplugin"
import "golang.org/x/sys/unix"

const (
	// Submatch index of the facility/level
	dmesgSmFacLev = 1
	// Submatch index of the timestamp, seconds part
	dmesgSmTsSec = 2
	// Submatch index of the timestamp, microseconds part
	dmesgSmTsUsec = 3
	// Submatch index of the actual message
	dmesgSmMsg = 4
	// Read all messages remaining in the ring buffer, placing then in the buffer pointed to  by  bufp. The
	// call reads the last len bytes from the log buffer (nondestructively), but will not read more than was
	// written into the buffer since the last "clear ring buffer" command (see command 5 below)). The call
	// returns the number of bytes read.
	sysActionReadAll int = 3
	// This command returns the total size of the kernel log buffer.
	sysActionSizeBuffer int = 10
)

func main () {
	var lastmsgtime int64
	undvolt := regexp.MustCompile(`Undervoltage detected`)
	tstamp := regexp.MustCompile(`\[(\d+)\.`)

	// Parse parameters
	var timewindow int
	var crit bool
	opt := getoptions.New()
	opt.BoolVar(&crit, "c", false)
	opt.IntVar(&timewindow, "t", 3600)
	opt.Parse(os.Args[1:])

	// Initialize Nagios module
	check := nagiosplugin.NewCheck()
	defer check.Finish()

	// Read dmesg
	dmesg, derr := ReadAll()
	if derr != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Error reading dmesg -> %s", derr.Error()))
		check.Finish()
	}

	// Convert bytes to lines of strings
	lines := byte_to_lines(dmesg)

	// Check for the most recent message regarding undervoltage, store its timestamp
	for _, line := range lines {
		if undvolt.MatchString(line) {
			match := tstamp.FindStringSubmatch(line)
			if len(match) == 2 {
				lastmsgtime, _ = strconv.ParseInt(match[1], 10, 64)
			}
		}
	}

	if lastmsgtime == 0 {
		// all good, no undervoltage reported
		check.AddResult(nagiosplugin.OK, "Voltage is perfect")
		check.Finish()
	} else {
		// Calculate if the message is in the timewindow we consider
		uptime, uterr := uptime.Get()
		if uterr != nil {
			check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Could not get system uptime -> %s", derr.Error()))
			check.Finish()
		}

		if lastmsgtime > int64(uptime.Seconds()) - int64(timewindow) {
			if crit {
				 check.AddResult(nagiosplugin.CRITICAL, "Undervoltage reported by hwmon")
				 check.Finish()
			} else {
				 check.AddResult(nagiosplugin.WARNING, "Undervoltage reported by hwmon")
				 check.Finish()
			}
		}
	}

	check.AddResult(nagiosplugin.OK, "Voltage had no recent issues")
	check.Finish()
//	uptime, _ := uptime.Get()
//	fmt.Println(uptime.Seconds())
}

func byte_to_lines (b []byte) ([]string) {
	return strings.Split(string(b), "\n")
}

func ReadAll() ([]byte, error) {
	// from https://github.com/vosst/csi/blob/767721bcc143/dmesg/entry.go#L77
	n, err := unix.Klogctl(sysActionSizeBuffer, nil)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to query size of log buffer [%s]", err))
	}

	b := make([]byte, n, n)

	m, err := unix.Klogctl(sysActionReadAll, b)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Failed to read messages from log buffer [%s]", err))
	}

	return b[:m], nil
}
