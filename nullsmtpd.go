package main

// This is a fake SMTP server
// It accepts any email and prints it to STDOUT
// It does not store messages

import "fmt"
import "log"
import "net"
import "os"
import "github.com/DavidGamba/go-getoptions"
import "github.com/mhale/smtpd"

func mailHandler(origin net.Addr, from string, to []string, data []byte) {
  fmt.Println("--- START OF MESSAGE ---")
  fmt.Print(string(data))
  fmt.Println("--- END OF MESSAGE ---")
}

func main() {
  var listen string = ":25"
  opt := getoptions.New()
  opt.StringVar(&listen, "l", listen)
  remaining, opterr := opt.Parse(os.Args[1:])
  if len(remaining) != 0 || opterr != nil {
	  fmt.Print(opt.Help())
	  os.Exit(1)
  }

  fmt.Printf("Listening on '%s'\n", listen)
  err := smtpd.ListenAndServe(listen, mailHandler, "NullSMTPD", "")
  if err != nil {
	  log.Println(err)
  }
}
