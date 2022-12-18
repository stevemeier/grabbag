package main

import "crypto/hmac"
import "crypto/md5"
import "crypto/tls"
import b64 "encoding/base64"
import "fmt"
import "io"
import "log"
import "net/smtp"
import "os"
import "strings"

import "github.com/DavidGamba/go-getoptions"

func main() {
	opt := getoptions.New()
	var username string
	var password string
	var server string
	var port string
	var insecure bool
	opt.StringVar(&server, "server", "", opt.Alias("s"), opt.Required())
	opt.StringVar(&port, "port", "25", opt.Alias("n"))
	opt.StringVar(&username, "username", "", opt.Alias("u"), opt.Required())
	opt.StringVar(&password, "password", "", opt.Alias("p"), opt.Required())
	opt.BoolVar(&insecure, "insecure", false)
	remaining, opterr := opt.Parse(os.Args[1:])
	if opterr != nil {
		log.Fatal(opterr)
	}
	if remaining != nil {
		log.Printf("Unhandled parameters: %v\n", remaining)
	}

	var authmethods string
	tlsconfig := &tls.Config{
		InsecureSkipVerify: insecure,
		ServerName: server,
	}

	log.Printf("Connecting to %s:%s\n", server, port)
	conn, connerr := smtp.Dial(server+":"+port)

	if connerr != nil {
		log.Fatal(connerr)
	}

	_, authmethods = conn.Extension("AUTH")
	log.Printf("Server supports: %s\n", authmethods)
	conn.Quit()
	conn.Close()

	for _, method := range strings.Split(authmethods, " ") {
		// Connect and read banner
		conn, connerr := smtp.Dial(server+":"+port)
		if connerr != nil {
			log.Fatalf("Could not connect: %s\n", connerr)
		}

		fmt.Printf("Next method: %s\n", method)
		if method == "LOGIN" {
			conn.Text.Writer.PrintfLine("AUTH LOGIN")
			read_server_response(conn)
			conn.Text.Writer.PrintfLine("%s", base64(username))
			read_server_response(conn)
			conn.Text.Writer.PrintfLine("%s", base64(password))
			code, response := read_server_response(conn)
			if code >= 200 && code < 300 {
				fmt.Println("AUTH LOGIN: SUCCESS\n")
			} else {
				fmt.Printf("AUTH LOGIN: FAILED -> %s\n", response)
			}
		}

		if method == "PLAIN" {
			tlserr := conn.StartTLS(tlsconfig)
			if tlserr != nil {
				log.Println(tlserr)
			} else {
				log.Println("STARTTLS OK")
			}
			plainauth := smtp.PlainAuth(username, username, password, server)
			perr := conn.Auth(plainauth)
			if perr != nil {
				fmt.Printf("AUTH PLAIN: FAILED -> %s\n", perr.Error())
			} else {
				fmt.Println("AUTH PLAIN: SUCCESS\n")
			}
		}

		if method == "CRAM-MD5" {
			cramauth := smtp.CRAMMD5Auth(username, password)
			cerr := conn.Auth(cramauth)
			if cerr != nil {
				fmt.Printf("AUTH CRAM-MD5: FAILED -> %s\n", cerr.Error())
			} else {
				fmt.Println("AUTH CRAM-MD5: SUCCESS\n")
			}
		}

		conn.Quit()
		conn.Close()
	}

}

func read_server_response (c *smtp.Client) (int, string) {
	code, message, _ := c.Text.ReadCodeLine(-1)
	return code, message
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

func hmac_md5_hex (challenge string, password string) string {
	hash := hmac.New(md5.New, []byte(password))
	io.WriteString(hash, challenge)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
