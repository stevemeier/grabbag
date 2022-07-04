package main

import "crypto/hmac"
import "crypto/md5"
import b64 "encoding/base64"
import "fmt"
import "io"
import "log"
import "net"
import "os"
import "regexp"
import "strings"
import "time"

import "github.com/davecgh/go-spew/spew"
import "github.com/DavidGamba/go-getoptions"

func main() {
	opt := getoptions.New()
	var username string
	var password string
	var server string
	var port string
	opt.StringVar(&server, "server", "", opt.Alias("s"), opt.Required())
	opt.StringVar(&port, "port", "25", opt.Alias("n"))
	opt.StringVar(&username, "username", "", opt.Alias("u"), opt.Required())
	opt.StringVar(&password, "password", "", opt.Alias("p"), opt.Required())
	remaining, opterr := opt.Parse(os.Args[1:])
	if opterr != nil {
		log.Fatal(opterr)
	}
	if remaining != nil {
		log.Printf("Unhandled parameters: %v\n", remaining)
	}

	var authmethods []string
	serverok := regexp.MustCompile("^2")

	log.Printf("Connecting to %s:%s\n", server, port)
	conn, connerr := net.Dial("tcp", server+":"+port)

	if connerr != nil {
		log.Fatal(connerr)
	}

	authmethods = get_auth_methods(conn)
	spew.Dump(authmethods)
	fmt.Fprintf(conn, "QUIT\r\n")
	conn.Close()

	for _, method := range authmethods {
		// Connect and read banner
		conn, connerr := net.Dial("tcp", server+":"+port)
		if connerr != nil {
			log.Fatalf("Could not connect: %s\n", connerr)
		}
		read_server_message(conn)

		fmt.Printf("Next method: %s\n", method)
		if method == "LOGIN" {
			fmt.Fprintf(conn, "AUTH LOGIN\r\n")
			read_server_message(conn)
			fmt.Fprintf(conn, "%s\r\n", base64(username))
			read_server_message(conn)
			fmt.Fprintf(conn, "%s\r\n", base64(password))
			response := read_server_message(conn)
			if serverok.Match(response) {
				fmt.Println("AUTH LOGIN: SUCCESS\n")
			} else {
				fmt.Printf("AUTH LOGIN: FAILED -> %s\n", response)
			}
			fmt.Fprintf(conn, "QUIT\r\n")
		}
		if method == "PLAIN" {
			fmt.Fprintf(conn, "AUTH PLAIN\r\n")
			read_server_message(conn)
			fmt.Fprintf(conn, "%s\r\n", base64(username+"\000"+username+"\000"+password))
			response := read_server_message(conn)
			if serverok.Match(response) {
				fmt.Println("AUTH PLAIN: SUCCESS\n")
			} else {
				fmt.Printf("AUTH PLAIN: FAILED -> %s\n", response)
			}
			fmt.Fprintf(conn, "QUIT\r\n")
		}
		if method == "CRAM-MD5" {
			fmt.Fprintf(conn, "AUTH CRAM-MD5\r\n")
			challenge := unbase64(string(read_server_message(conn)[4:]))
			fmt.Fprintf(conn, "%s\r\n", base64(username+" "+hmac_md5_hex(challenge,password)))
			response := read_server_message(conn)
			if serverok.Match(response) {
				fmt.Println("AUTH CRAM-MD5: SUCCESS\n")
			} else {
				fmt.Printf("AUTH CRAM-MD5: FAILED -> %s\n", response)
			}
			fmt.Fprintf(conn, "QUIT\r\n")
		}
		time.Sleep(1 * time.Second)
		conn.Close()
	}
}

func read_server_message (c net.Conn) ([]byte) {
	buffer := make([]byte, 1024)
	_, err := c.Read(buffer)
	if err != nil {
		return []byte{}
	}

	return buffer
}

func error_out (m []byte) () {
	log.Fatal(string(m))
}

func base64 (s string) (string) {
	return b64.StdEncoding.EncodeToString([]byte(s))
}

func unbase64 (s string) (string) {
	result, _ := b64.StdEncoding.DecodeString(s)
	return string(result)
}

func get_auth_methods (c net.Conn) ([]string) {
	var result []string

	serverok := regexp.MustCompile("^2")
	authre := regexp.MustCompile("(AUTH.)(.*)\\r")

	read_server_message(c)
	fmt.Fprintf(c, "EHLO example.com\r\n")
	response := read_server_message(c)
	if serverok.Match(response) {
		authdata := authre.FindStringSubmatch(string(response))
		if len(authdata) == 0 { error_out([]byte("Authentication not supported")) }
		result = strings.Split(authdata[2], " ")
	}

	return result
}

func hmac_md5_hex (challenge string, password string) string {
	hash := hmac.New(md5.New, []byte(password))
	io.WriteString(hash, challenge)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
