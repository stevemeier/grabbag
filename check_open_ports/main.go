package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/Ullaakut/nmap/v3"
)

func main() {
	var host string
	var tcpPorts string
	var udpPorts string
	var nmapBinary string = "/usr/bin/nmap"
	var nmapAltBinary string

	flag.StringVar(&host, "host", "", "Host")
	flag.StringVar(&tcpPorts, "tcp", "", "TCP ports which should be open")
	flag.StringVar(&udpPorts, "udp", "", "UDP ports which should be open")
	flag.StringVar(&nmapAltBinary, "bin", "", "Path of nmap binary to use")

	flag.Parse()

	// Default or custom nmap path
	if nmapAltBinary != "" {
		nmapBinary = nmapAltBinary
	}

	// Check nmap binary
	if _, err := os.Stat(nmapBinary); os.IsNotExist(err) {
		fmt.Printf("UNKNOWN - nmap binary not found or not executable at %s\n", nmapBinary)
		os.Exit(3) // Nagios UNKNOWN status
	}

	// Run nmap
	ctx, cancel := context.WithTimeout(context.Background(), 55 * time.Second)
	defer cancel()

	scanner, _ := nmap.NewScanner(
		ctx,
		nmap.WithTargets(host),
		nmap.WithPorts(tcpPorts),
		nmap.WithUDPDiscovery(udpPorts),
		nmap.WithBinaryPath(nmapBinary),
		nmap.WithSkipHostDiscovery(),
	)
	result, warnings, err := scanner.Run()

	if len(*warnings) > 0 {
		fmt.Printf("WARNING - nmap returned %d warnings\n", len(*warnings))
		os.Exit(1) // Nagios WARNING status
	}

	if err != nil {
		fmt.Printf("UNKNOWN - nmap exited with error: %v\n", err)
		os.Exit(3) // Nagios UNKNOWN status
	}

	// Check open ports
	notOpen := ComparePorts(tcpPorts, udpPorts, result)

	if len(notOpen) > 0 {
		fmt.Printf("CRITICAL - These ports are not open: %s\n", strings.Join(notOpen, ", "))
		os.Exit(2) // Nagios CRITICAL status
	} else {
		fmt.Printf("OK - All ports are open\n")
		os.Exit(0) // Nagios OK status
	}
}

func ComparePorts (tcpPorts string, udpPorts string, nmaprun *nmap.Run) ([]string) {
	var result []string

	// Build a map of expected ports
	allports := make(map[string]bool)
	if len(tcpPorts) > 0 {
		for _, tcpPort := range strings.Split(tcpPorts, ",") {
			allports[tcpPort+"/tcp"] = false
		}
	}
	if len(udpPorts) > 0 {
		for _, udpPort := range strings.Split(udpPorts, ",") {
			allports[udpPort+"/udp"] = false
		}
	}

	for _, host := range nmaprun.Hosts {
		for _, port := range host.Ports {
			if port.State.String() != "open" { continue }
			allports[strconv.FormatUint(uint64(port.ID), 10)+"/"+port.Protocol] = true
		}
	}

	for k, v := range allports {
		if !v { result = append(result, k) }
	}

	return result
}
