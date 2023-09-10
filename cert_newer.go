package main

// Takes to certificates and exists 0 if the second one is an
// in-place "upgrade" for the first

// Return Codes:
// 0 - SUCCESS
// 1 - New certificate is not a possible replacement
// 2 - Not enough parameters
// 3 - Certificate files could not be read (not found, not readable)
// 4 - Certificate files could not be parsed (wrong format)

import "crypto/rsa"
import "crypto/x509"
import "encoding/pem"
import "fmt"
import "io/ioutil"
import "os"
import "reflect"
import "time"

func main() {
	// Make sure we have two arguments to work with
	if len(os.Args) < 2 {
		os.Exit(2)
	}

	// Read the certificate files
	certdata1, ioerr1 := ioutil.ReadFile(os.Args[1])
	certdata2, ioerr2 := ioutil.ReadFile(os.Args[2])
	if ioerr1 != nil || ioerr2 != nil {
		os.Exit(3)
	}

	// Default DER first, then PEM
	cert1, cerr1 := x509.ParseCertificate(certdata1)
	if cerr1 != nil {
		cert1, cerr1 = x509.ParseCertificate(pem_to_der(certdata1))
	}
	cert2, cerr2 := x509.ParseCertificate(certdata2)
	if cerr2 != nil {
		cert2, cerr2 = x509.ParseCertificate(pem_to_der(certdata2))
	}

	// Check if Certificates were parsed
	if cerr1 != nil || cerr2 != nil {
		os.Exit(4)
	}

	// Success, if the following criteria are met:
	// - Identical Common Name and Subject Alternative Names
	// - Public Keys are identical
	// - New certificate expires after old certificate
	// - New certificate is valid (doesn't start in the future)
	if reflect.DeepEqual(cert1.Subject, cert2.Subject) &&
	   cert1.PublicKey.(*rsa.PublicKey).Equal(cert2.PublicKey.(*rsa.PublicKey)) &&
	   cert2.NotAfter.After(cert1.NotAfter) &&
	   cert2.NotBefore.Before(time.Now()) {
		os.Exit(0)
	}

	// Default exit 1
	os.Exit(1)
}

func pem_to_der (in []byte) ([]byte) {
	pemdata, _ := pem.Decode(in)
	return pemdata.Bytes
}
