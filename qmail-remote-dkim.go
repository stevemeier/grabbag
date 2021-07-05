package main

import "bytes"
import "errors"
import "fmt"
import "io/ioutil"
import "log/syslog"
import "os"
import "os/exec"
import "strings"
import "syscall"
import dkim "github.com/toorop/go-dkim"
import "io"
import "github.com/emersion/go-smtp"

// Exit codes
// 1 - Failed to read message from stdin
// 2 - Failed to read key for domain
// 3 - Failed to sign email
// 4 - Failed to set up syslog

func main() {
	// Setup syslog
	// mail = 2, info = 6
	// priority = 8 * 2 + 6 = 22
	if env_defined("QBOX_DEBUG") { fmt.Println("START: Setting up syslog") }
	syslog, err := syslog.New(22, "qmail-remote-dkim")
	if err != nil {
		os.Exit(4)
	}
	if env_defined("QBOX_DEBUG") { fmt.Println("END: Setting up syslog") }

	// Read message from stdin
	if env_defined("QBOX_DEBUG") { fmt.Println("START: Reading from stdin") }
	email, err := read_from_stdin()
	if err != nil {
		os.Exit(1)
	}
	if env_defined("QBOX_DEBUG") { fmt.Println("END: Reading from stdin") }

	// Extract sender domain
	domain := domain_of(os.Args[2])

	// Sign email, if key is available
	if env_defined("QBOX_DEBUG") { fmt.Println("START: Signing part") }
	if file_exists("/var/qmail/control/dkim/"+domain+".pem") {
		key, keyerr := ioutil.ReadFile("/var/qmail/control/dkim/"+domain+".pem")
		if keyerr != nil {
			os.Exit(2)
		}

		options := dkim.NewSigOptions()
		options.PrivateKey = key
		options.Domain = domain
		options.Selector = "dkim"
		options.Headers = []string{"from", "date", "subject"}
		options.AddSignatureTimestamp = true
		options.Canonicalization = "relaxed/relaxed"

		syslog.Write([]byte("Signing message from "+os.Args[2]))
		err := dkim.Sign(&email, options)
		if err != nil {
			os.Exit(3)
		}
	} else {
		syslog.Write([]byte("Passing through message from "+os.Args[2]))
	}
	if env_defined("QBOX_DEBUG") { fmt.Println("END: Signing part") }

	if env_defined("QBOX_DEBUG") { fmt.Printf("%s", email) }

	// Call original qmail-remote with signed message
//	if env_defined("QBOX_DEBUG") { fmt.Println("START: Forking qmail-remote.orig") }
//	output, exitcode, _ := sysexec("/var/qmail/bin/qmail-remote.orig", os.Args[1:], email)
//	if env_defined("QBOX_DEBUG") { fmt.Println("END: Forking qmail-remote.orig") }
//	fmt.Println(string(output))

        deliveryerr := smtp_delivery(os.Args[1], os.Args[2], os.Args[3:], email)
	if deliveryerr == nil {
	  fmt.Print("r"+os.Args[2]+" accepted the message\000")
	}

	// qmail-remote always exits zero according to man-page
	// but this doesn't cost us anything
	os.Exit(exitcode)
}

func sysexec (command string, args []string, input []byte) ([]byte, int, error) {
        var output bytes.Buffer

        if !file_exists(command) {
                return nil, 111, errors.New("command not found")
        }

        if !is_executable(command) {
                return nil, 111, errors.New("command not executable")
        }

        cmd := exec.Command(command, args...)
        cmd.Stdin = bytes.NewBuffer(input)
        cmd.Stdout = &output
        err := cmd.Run()

        exitcode := 0
        if exitError, ok := err.(*exec.ExitError); ok {
                exitcode = exitError.ExitCode()
        }

        return output.Bytes(), exitcode, err
}

func read_from_stdin () ([]byte, error) {
        var message []byte
	message, err := ioutil.ReadAll(os.Stdin)
	return message, err
}

func is_executable (file string) bool {
        stat, err := os.Stat(file)
        if err != nil {
                return false
        }

        // These calls return uint32 by default while
        // os.Get?id returns int. So we have to change one
        fileuid := int(stat.Sys().(*syscall.Stat_t).Uid)
        filegid := int(stat.Sys().(*syscall.Stat_t).Gid)

        if (os.Getuid() == fileuid) { return stat.Mode()&0100 != 0 }
        if (os.Getgid() == filegid) { return stat.Mode()&0010 != 0 }
        return stat.Mode()&0001 != 0
}

func file_exists(filename string) bool {
        info, err := os.Stat(filename)
        if os.IsNotExist(err) {
                return false
        }

        return !info.IsDir()
}

func domain_of (address string) string {
        addrparts := strings.Split(address, "@")
        if len(addrparts) == 2 {
                return addrparts[1]
        } else {
                return "default"
        }
}

func env_defined(key string) bool {
        _, exists := os.LookupEnv(key)
        return exists
}

func smtp_delivery (host string, sender string, to []string, email io.Reader) (error) {
  err := smtp.SendMail(host+":25", nil, sender, to, email)
  return err
}
