package main

import "bufio"
import "fmt"
import "os"
import "github.com/glaslos/tlsh"

func main() {
  // stdin mode
  if len(os.Args) <= 1 {
    reader := bufio.NewReader(os.Stdin)
    obj, err := tlsh.HashReader(reader)
    if err != nil {
      fmt.Println(err)
      os.Exit(1)
    }
    fmt.Printf("%s\t%s\n", obj.String(), "-")
    os.Exit(0)
  }

  // parameters mode
  var errcount int
  for _, file := range os.Args[1:] {
    obj, err := tlsh.HashFilename(file)
    if err != nil {
      fmt.Println(err)
      errcount = 1
    } else {
      fmt.Printf("%s\t%s\n", obj.String(), file)
    }
  }
  os.Exit(errcount)

}
