package main

import (
	"crypto/tls"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	// Written mostly by ChatGPT :-D
	// Added reading URL from Argv

	if len(os.Args) == 1 {
		fmt.Fprintf(os.Stderr, "Missing server name\n")
		os.Exit(1)
	}
	serverAddr := os.Args[1]

	// If user did not provide a port, add :443
	if !strings.Contains(serverAddr, ":") {
		serverAddr = serverAddr + ":443"
	}

	// Set a timeout for the connection
	dialer := net.Dialer{
		Timeout:   5 * time.Second, // Adjust the timeout as needed
		KeepAlive: 0,
	}

	// Dial the server with timeout
	conn, err := dialer.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server: %s\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Set up a TLS configuration with InsecureSkipVerify set to true
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Perform a TLS handshake with a timeout
	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.SetDeadline(time.Now().Add(10 * time.Second)) // Adjust the timeout as needed
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting deadline: %s\n", err)
		os.Exit(2)
	}

	err = tlsConn.Handshake()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TLS handshake error: %s\n", err)
		os.Exit(2)
	}
	defer tlsConn.Close()

	// Get the server's TLS certificate
	leafCert := tlsConn.ConnectionState().PeerCertificates[0]

	// Convert the leaf certificate to PEM format
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafCert.Raw})

	// Print the PEM-formatted certificate
	fmt.Print(string(pemCert))
}

