package main

import "bufio"
import "fmt"
import "log"
import "net"
import "regexp"
import "strings"
import "os"
import "sort"
import "strconv"

import "github.com/c-robinson/iplib"
//import "github.com/davecgh/go-spew/spew"

func main() {
	ip4world := make(map[uint32][]string)
	ip6regexp, _ := regexp.Compile(":")
	curlyregexp, _ := regexp.Compile("{")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {

		input := strings.Split(scanner.Text(), ` `)
		if len(input) < 2 {
			continue
		}

		// Ignore IPv6 addresses
		if ip6regexp.MatchString(input[0]) {
			continue
		}

		// Ignore entries with curly brackets
		if curlyregexp.MatchString(input[1]) {
			continue
		}

		i, err := strconv.ParseInt(input[1], 10, 64)
		if err == nil {
			// Ignore 16-bit private AS numbers
			if i >= 64512 && i <= 65535 {
				continue
			}
			// Ignore 32-bit private AS numbers
			if i >= 4200000000 && i <= 4294967294 {
				continue
			}
		}

		_, ipna, _ := iplib.ParseCIDR(input[0])

		if ipna.Count() <= 256 {
			continue
		}

		subnets, err := ipna.Subnet(24)
		if err != nil {
			log.Fatal(err)
		}

		for _, prefix := range subnets {
//			fmt.Println(iplib_to_string(prefix))
			ipa := net.ParseIP(iplib_to_string(prefix))
//			fmt.Println(iplib.IP4ToUint32(ipa))
			ipkey := iplib.IP4ToUint32(ipa)
			ip4world[ipkey] = append(ip4world[ipkey], input[1])
		}
	}

	for network := range ip4world {
		// if announced by more than on AS, make sure it's unique
		if len(ip4world[network]) > 1 {
			ip4world[network] = unique(ip4world[network])
		}
		// If we get here, it's really two different AS!
		if len(ip4world[network]) > 1 {
//			fmt.Println(network)
			fmt.Print(iplib.Uint32ToIP4(network))
			fmt.Print("\t")
			fmt.Print(strings.Join(ip4world[network], "\t")+"\n")
//			spew.Dump(ip4world[network])
		}
	}
}

func iplib_to_string (obj iplib.Net) (string) {
//	len, _ := obj.Mask.Size()
//	return fmt.Sprintf("%s/%d", obj.NetworkAddress(), len)
	return fmt.Sprintf("%s", obj.NetworkAddress())
}

func unique(slice []string) []string {
    keys := make(map[string]bool)
    list := []string{}
    for _, entry := range slice {
        if _, value := keys[entry]; !value {
            keys[entry] = true
            list = append(list, entry)
        }
    }
    sort.Strings(list)
    return list
}
