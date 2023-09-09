package main
import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	"golang.org/x/exp/slices"
	"github.com/DavidGamba/go-getoptions"
	"github.com/olorin/nagiosplugin"
)

func main() {
        // Read command-line options
	var host string
	var port string
	var reqauth bool
	var password string
        var goodstates []string
	var debug bool
	var justwarn bool
        opt := getoptions.New()
        opt.StringVar(&host, "host", "", opt.Alias("h"), opt.Required())
        opt.StringVar(&port, "port", "", opt.Alias("p"), opt.Required())
        opt.BoolVar(&reqauth, "auth", false, opt.Alias("S"))
        opt.StringVar(&password, "password", "", opt.Alias("P"))
	opt.StringSliceVar(&goodstates, "accept", 1, 100, opt.Alias("a"))
	opt.BoolVar(&debug, "debug", false)
	opt.BoolVar(&justwarn, "warn", false, opt.Alias("w"))
        opt.Parse(os.Args[1:])
        if len(os.Args[1:]) == 0 {
                fmt.Print(opt.Help())
                os.Exit(1)
        }

	// Initialize Nagios
        check := nagiosplugin.NewCheck()
        defer check.Finish()

	// Establish the TCP connection
	conn, err := net.DialTimeout("tcp", host+":"+port, 5 * time.Second)
	if err != nil {
		check.AddResult(nagiosplugin.UNKNOWN, err.Error())
		check.Finish()
	}
	defer conn.Close()

	// At this point we should receive a banner from the server
	// It's either the password prompt or the `shell`
	banner := lazyread(conn)

	// Check if server is requiring a password
	pwrequired, _ := regexp.MatchString(`(?i)password`, banner)
	if pwrequired {
		if len(password) == 0 {
			check.AddResult(nagiosplugin.UNKNOWN, "Password required but not provided")
			check.Finish()
	    	}
	    
		// Send the password
		lazywrite(conn, fmt.Sprintf("%s\n", password))
		// Check server response to password
		if lazyread(conn) == banner {
			check.AddResult(nagiosplugin.UNKNOWN, "Password incorrect")
			check.Finish()
		}
		
		// Perform one more read to get to the prompt
		lazyread(conn)
	}

	// Send `state` command
	lazywrite(conn, "state\n")
	state := lazyread(conn)
	state = parse_state(state)

	// Check if parsing the state worked
	if state == "UNKNOWN" {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Could not parse server state", state))
		check.Finish()
	}

	// Check if state is `CONNECTED` or in the list of good states
	// If yes, everything is great, job done
	if state == "CONNECTED" ||
	   slices.Contains(goodstates, state) {
		check.AddResult(nagiosplugin.OK, fmt.Sprintf("Status %s", state))
		check.Finish()
	}

	// If we get here, we are in a bad state (not connected, not good state)
	// By default, this is critical. Parameter can downgrade this to warning
	if justwarn {
		check.AddResult(nagiosplugin.WARNING, fmt.Sprintf("Status %s", state))
	} else {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("Status %s", state))
	}
	check.Finish()
}

func lazyread (c net.Conn) (string) {
	msg := make([]byte, 1024)
	_, err := c.Read(msg)

	if err == nil {
		return string(msg)
	}

	return ""
}

func lazywrite (c net.Conn, s string) (bool) {
	_, err := c.Write([]byte(s))

	return err == nil
}

func parse_state (s string) (string) {
	fields := strings.Split(s, ",")
	if len(fields) < 2 {
		return "UNKNOWN"
	}

	return fields[1]
}
