package main

import "bufio"
import "bytes"
import "errors"
import "fmt"
import "io/ioutil"
import "os"
import "os/exec"
import "strings"
import "syscall"
import dkim "github.com/toorop/go-dkim"

// Exit codes
// 1 - Failed to read message from stdin
// 2 - Failed to read key for domain
// 3 - Failed to sign email

func main() {
	// Read message from stdin
	email, err := read_from_stdin()
	if err != nil {
		os.Exit(1)
	}

	// Extract sender domain
	domain := domain_of(os.Args[2])

	// Sign email, if key is available
	if file_exists("/var/qmail/control/dkim/"+domain+".pem") {
		key, keyerr := ioutil.ReadFile("/var/qmail/control/dkim/"+domain+".pem")
		if keyerr != nil {
			os.Exit(2)
		}

		options := dkim.NewSigOptions()
		options.PrivateKey = key
		options.Domain = domain
		options.Selector = "dkim"

		err := dkim.Sign(&email, options)
		if err != nil {
			os.Exit(3)
		}
	}

	// Call original qmail-remote with signed message
	output, exitcode, _ := sysexec("/var/qmail/bin/qmail-remote.orig", os.Args[1:], email)
	fmt.Println(string(output))

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
        scanner := bufio.NewScanner(os.Stdin)
        if scanner.Err() != nil {
		return []byte(``), scanner.Err()
        }

        var message string
        for scanner.Scan() {
                message = message + scanner.Text() + "\n"
        }

        return []byte(message), nil
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
