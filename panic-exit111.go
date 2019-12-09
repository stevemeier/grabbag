package main

import "fmt"
import "os"

func main() {
	defer func() {
		if err := recover(); err != nil {
			os.Exit(111)
		}
	}()

	var test []string
	fmt.Println(test[1])
}
