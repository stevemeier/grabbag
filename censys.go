package main

import "crypto/sha256"
import "crypto/x509"
import "crypto/x509/pkix"
import "encoding/asn1"
import "encoding/pem"
import "math/big"

import "encoding/json"
import "fmt"
import "io/ioutil"
import "log"
import "os"
import "net"
import "net/http"
import "net/url"
import "strings"
import "time"

import "github.com/DavidGamba/go-getoptions"

// Source: https://golang.org/pkg/crypto/x509/#Certificate
type Certificate struct {
        Raw                     []byte // Complete ASN.1 DER content (certificate, signature algorithm and signature).
        RawTBSCertificate       []byte // Certificate part of raw ASN.1 DER content.
        RawSubjectPublicKeyInfo []byte // DER encoded SubjectPublicKeyInfo.
        RawSubject              []byte // DER encoded Subject
        RawIssuer               []byte // DER encoded Issuer

        Signature          []byte
//        SignatureAlgorithm SignatureAlgorithm

//        PublicKeyAlgorithm PublicKeyAlgorithm
        PublicKey          interface{}

        Version             int
        SerialNumber        *big.Int
        Issuer              pkix.Name
        Subject             pkix.Name
        NotBefore, NotAfter time.Time // Validity bounds.
//        KeyUsage            KeyUsage

        // Extensions contains raw X.509 extensions. When parsing certificates,
        // this can be used to extract non-critical extensions that are not
        // parsed by this package. When marshaling certificates, the Extensions
        // field is ignored, see ExtraExtensions.
        Extensions []pkix.Extension // Go 1.2

        // ExtraExtensions contains extensions to be copied, raw, into any
        // marshaled certificates. Values override any extensions that would
        // otherwise be produced based on the other fields. The ExtraExtensions
        // field is not populated when parsing certificates, see Extensions.
        ExtraExtensions []pkix.Extension // Go 1.2

        // UnhandledCriticalExtensions contains a list of extension IDs that
        // were not (fully) processed when parsing. Verify will fail if this
        // slice is non-empty, unless verification is delegated to an OS
        // library which understands all the critical extensions.
        //
        // Users can access these extensions using Extensions and can remove
        // elements from this slice if they believe that they have been
        // handled.
        UnhandledCriticalExtensions []asn1.ObjectIdentifier // Go 1.5

//        ExtKeyUsage        []ExtKeyUsage           // Sequence of extended key usages.
        UnknownExtKeyUsage []asn1.ObjectIdentifier // Encountered extended key usages unknown to this package.

        // BasicConstraintsValid indicates whether IsCA, MaxPathLen,
        // and MaxPathLenZero are valid.
        BasicConstraintsValid bool
        IsCA                  bool

        // MaxPathLen and MaxPathLenZero indicate the presence and
        // value of the BasicConstraints' "pathLenConstraint".
        //
        // When parsing a certificate, a positive non-zero MaxPathLen
        // means that the field was specified, -1 means it was unset,
        // and MaxPathLenZero being true mean that the field was
        // explicitly set to zero. The case of MaxPathLen==0 with MaxPathLenZero==false
        // should be treated equivalent to -1 (unset).
        //
        // When generating a certificate, an unset pathLenConstraint
        // can be requested with either MaxPathLen == -1 or using the
        // zero value for both MaxPathLen and MaxPathLenZero.
        MaxPathLen int
        // MaxPathLenZero indicates that BasicConstraintsValid==true
        // and MaxPathLen==0 should be interpreted as an actual
        // maximum path length of zero. Otherwise, that combination is
        // interpreted as MaxPathLen not being set.
        MaxPathLenZero bool // Go 1.4

        SubjectKeyId   []byte
        AuthorityKeyId []byte

        // RFC 5280, 4.2.2.1 (Authority Information Access)
        OCSPServer            []string // Go 1.2
        IssuingCertificateURL []string // Go 1.2

        // Subject Alternate Name values. (Note that these values may not be valid
        // if invalid values were contained within a parsed certificate. For
        // example, an element of DNSNames may not be a valid DNS domain name.)
        DNSNames       []string
        EmailAddresses []string
        IPAddresses    []net.IP // Go 1.1
        URIs           []*url.URL // Go 1.10

        // Name constraints
        PermittedDNSDomainsCritical bool // if true then the name constraints are marked critical.
        PermittedDNSDomains         []string
        ExcludedDNSDomains          []string // Go 1.9
        PermittedIPRanges           []*net.IPNet // Go 1.10
        ExcludedIPRanges            []*net.IPNet // Go 1.10
        PermittedEmailAddresses     []string // Go 1.10
        ExcludedEmailAddresses      []string // Go 1.10
        PermittedURIDomains         []string // Go 1.10
        ExcludedURIDomains          []string // Go 1.10

        // CRL Distribution Points
        CRLDistributionPoints []string // Go 1.2

        PolicyIdentifiers []asn1.ObjectIdentifier
}

type SearchResult struct {
	Metadata struct {
		BackendTime int    `json:"backend_time"`
		Count       int    `json:"count"`
		Page        int    `json:"page"`
		Pages       int    `json:"pages"`
		Query       string `json:"query"`
	} `json:"metadata"`
	Results []struct {
		ParsedFingerprintSha256 string `json:"parsed.fingerprint_sha256"`
		ParsedIssuerDn          string `json:"parsed.issuer_dn"`
		ParsedSubjectDn         string `json:"parsed.subject_dn"`
	} `json:"results"`
	Status string `json:"status"`
}

type CertificateDetails struct {
	Ct struct {
		CloudflareNimbus2021 struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"cloudflare_nimbus_2021"`
		ComodoDodo struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"comodo_dodo"`
		ComodoMammoth struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"comodo_mammoth"`
		ComodoSabre struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"comodo_sabre"`
		DigicertCt2 struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"digicert_ct2"`
		GdcaLog struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"gdca_log"`
		GdcaLog2 struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"gdca_log2"`
		GoogleArgon2021 struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"google_argon_2021"`
		GoogleAviator struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"google_aviator"`
		GoogleIcarus struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"google_icarus"`
		GooglePilot struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"google_pilot"`
		GoogleRocketeer struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"google_rocketeer"`
		NorduCtPlausible struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"nordu_ct_plausible"`
		VenafiAPICtlog struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"venafi_api_ctlog"`
		VenafiAPICtlogGen2 struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"venafi_api_ctlog_gen2"`
		WosignCtlog struct {
			AddedToCtAt  time.Time `json:"added_to_ct_at"`
			CtToCensysAt time.Time `json:"ct_to_censys_at"`
			Index        int       `json:"index"`
		} `json:"wosign_ctlog"`
	} `json:"ct"`
	Metadata struct {
		ParseStatus   string `json:"parse_status"`
		ParseVersion  int    `json:"parse_version"`
		PostProcessed bool   `json:"post_processed"`
		SeenInScan    bool   `json:"seen_in_scan"`
		Source        string `json:"source"`
		UpdatedAt     string `json:"updated_at"`
	} `json:"metadata"`
	ParentSpkiSubjectFingerprint string        `json:"parent_spki_subject_fingerprint"`
	Parents                      []interface{} `json:"parents"`
	Parsed                       struct {
		Extensions struct {
			AuthorityInfoAccess struct {
				IssuerUrls []string `json:"issuer_urls"`
				OcspUrls   []string `json:"ocsp_urls"`
			} `json:"authority_info_access"`
			AuthorityKeyID   string `json:"authority_key_id"`
			BasicConstraints struct {
				IsCa       bool `json:"is_ca"`
				MaxPathLen int  `json:"max_path_len"`
			} `json:"basic_constraints"`
			CertificatePolicies []struct {
				ID  string   `json:"id"`
				Cps []string `json:"cps,omitempty"`
			} `json:"certificate_policies"`
			CrlDistributionPoints []string `json:"crl_distribution_points"`
			KeyUsage              struct {
				CertificateSign  bool `json:"certificate_sign"`
				CrlSign          bool `json:"crl_sign"`
				DigitalSignature bool `json:"digital_signature"`
				Value            int  `json:"value"`
			} `json:"key_usage"`
			SubjectKeyID string `json:"subject_key_id"`
		} `json:"extensions"`
		FingerprintMd5    string `json:"fingerprint_md5"`
		FingerprintSha1   string `json:"fingerprint_sha1"`
		FingerprintSha256 string `json:"fingerprint_sha256"`
		Issuer            struct {
			CommonName   []string `json:"common_name"`
			Organization []string `json:"organization"`
		} `json:"issuer"`
		IssuerDn     string `json:"issuer_dn"`
		Redacted     bool   `json:"redacted"`
		SerialNumber string `json:"serial_number"`
		Signature    struct {
			SelfSigned         bool `json:"self_signed"`
			SignatureAlgorithm struct {
				Name string `json:"name"`
				Oid  string `json:"oid"`
			} `json:"signature_algorithm"`
			Valid bool   `json:"valid"`
			Value string `json:"value"`
		} `json:"signature"`
		SignatureAlgorithm struct {
			Name string `json:"name"`
			Oid  string `json:"oid"`
		} `json:"signature_algorithm"`
		SpkiSubjectFingerprint string `json:"spki_subject_fingerprint"`
		Subject                struct {
			CommonName   []string `json:"common_name"`
			Country      []string `json:"country"`
			Organization []string `json:"organization"`
		} `json:"subject"`
		SubjectDn      string `json:"subject_dn"`
		SubjectKeyInfo struct {
			FingerprintSha256 string `json:"fingerprint_sha256"`
			KeyAlgorithm      struct {
				Name string `json:"name"`
			} `json:"key_algorithm"`
			RsaPublicKey struct {
				Exponent int    `json:"exponent"`
				Length   int    `json:"length"`
				Modulus  string `json:"modulus"`
			} `json:"rsa_public_key"`
		} `json:"subject_key_info"`
		TbsFingerprint     string `json:"tbs_fingerprint"`
		TbsNoctFingerprint string `json:"tbs_noct_fingerprint"`
		ValidationLevel    string `json:"validation_level"`
		Validity           struct {
			End    time.Time `json:"end"`
			Length int       `json:"length"`
			Start  time.Time `json:"start"`
		} `json:"validity"`
		Version int `json:"version"`
	} `json:"parsed"`
	Precert    bool     `json:"precert"`
	Raw        string   `json:"raw"`
	Tags       []string `json:"tags"`
	Validation struct {
		Apple struct {
			Blacklisted     bool       `json:"blacklisted"`
			HadTrustedPath  bool       `json:"had_trusted_path"`
			InRevocationSet bool       `json:"in_revocation_set"`
			Parents         []string   `json:"parents"`
			Paths           [][]string `json:"paths"`
			TrustedPath     bool       `json:"trusted_path"`
			Type            string     `json:"type"`
			Valid           bool       `json:"valid"`
			WasValid        bool       `json:"was_valid"`
			Whitelisted     bool       `json:"whitelisted"`
		} `json:"apple"`
		GoogleCtPrimary struct {
			Blacklisted     bool       `json:"blacklisted"`
			HadTrustedPath  bool       `json:"had_trusted_path"`
			InRevocationSet bool       `json:"in_revocation_set"`
			Parents         []string   `json:"parents"`
			Paths           [][]string `json:"paths"`
			TrustedPath     bool       `json:"trusted_path"`
			Type            string     `json:"type"`
			Valid           bool       `json:"valid"`
			WasValid        bool       `json:"was_valid"`
			Whitelisted     bool       `json:"whitelisted"`
		} `json:"google_ct_primary"`
		Microsoft struct {
			Blacklisted     bool       `json:"blacklisted"`
			HadTrustedPath  bool       `json:"had_trusted_path"`
			InRevocationSet bool       `json:"in_revocation_set"`
			Parents         []string   `json:"parents"`
			Paths           [][]string `json:"paths"`
			TrustedPath     bool       `json:"trusted_path"`
			Type            string     `json:"type"`
			Valid           bool       `json:"valid"`
			WasValid        bool       `json:"was_valid"`
			Whitelisted     bool       `json:"whitelisted"`
		} `json:"microsoft"`
		Nss struct {
			Blacklisted     bool       `json:"blacklisted"`
			HadTrustedPath  bool       `json:"had_trusted_path"`
			InRevocationSet bool       `json:"in_revocation_set"`
			Parents         []string   `json:"parents"`
			Paths           [][]string `json:"paths"`
			TrustedPath     bool       `json:"trusted_path"`
			Type            string     `json:"type"`
			Valid           bool       `json:"valid"`
			WasValid        bool       `json:"was_valid"`
			Whitelisted     bool       `json:"whitelisted"`
		} `json:"nss"`
	} `json:"validation"`
	Zlint struct {
		ErrorsPresent   bool `json:"errors_present"`
		FatalsPresent   bool `json:"fatals_present"`
		NoticesPresent  bool `json:"notices_present"`
		Version         int  `json:"version"`
		WarningsPresent bool `json:"warnings_present"`
	} `json:"zlint"`
}

func main () {
	// Parse arguments
	opt := getoptions.New()

	var cert string
	var fullchain string
	var debug bool
	opt.StringVar(&cert, "cert", "", opt.Required())
	opt.StringVar(&fullchain, "fullchain", "")
        opt.BoolVar(&debug, "debug", false)
	remaining, err := opt.Parse(os.Args[1:])

	// Handle empty or unknown options
	if len(os.Args[1:]) == 0 {
		log.Print(opt.Help())
		os.Exit(1)
        }
	if err != nil {
		log.Fatalf("Could not parse options: %s\n", err)
		os.Exit(1)
	}
	if len(remaining) > 0 {
		log.Fatalf("Unsupported parameter: %s\n", remaining)
		os.Exit(1)
	}

	// read username and password from ENV
	username := os.Getenv("CENSYS_APIID")
	password := os.Getenv("CENSYS_SECRET")
	if username == "" {
		log.Fatal("Please set $CENSYS_APIID to your API ID\n")
		os.Exit(1)
	}
	if password == "" {
		log.Fatal("Please set $CENSYS_SECRET to your API secret\n")
		os.Exit(1)
	}

	// Parse certificate
	subject, notbefore, sha256fp := get_cert_details(cert)
	if debug {
		fmt.Printf("Current cert subject:     %s\n", subject)
		fmt.Printf("Current cert start date:  %s\n", notbefore)
		fmt.Printf("Current cert fingerprint: %s\n", sha256fp)
	}

	// Create HTTP client
	client := &http.Client{}

	// Search for matching certificates
	req, err := http.NewRequest("POST", "https://censys.io/api/v1/search/certificates", strings.NewReader(`{"query":"parsed.subject_dn: ` + subject + ` and validation.nss.valid: true"}`) )
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(username, password)
	resp, _ := client.Do(req)

	if resp.StatusCode >= 400 {
		log.Fatalf("API search failed with response: %s\n", resp.Status)
		os.Exit(3)
	}

	body, err := ioutil.ReadAll(resp.Body)

	var searchresult SearchResult
	err = json.Unmarshal(body, &searchresult)
	if err != nil {
		log.Fatalf("Could not parse search result: %s\n", err)
		os.Exit(3)
	}

	if searchresult.Status != "ok" {
		log.Fatalf("Search result status is: %s", searchresult.Status)
		os.Exit(3)
	}

	// Iterate over returned certificates
	// curl https://censys.io/api/v1/view/certificates/7764a7e399d7a12c24a4a9f4115cca051b7c7ddf4be8e0c702255a20d9567a9c
	// Find latest cert
	var replacement CertificateDetails

	var candidates []string
	for _, certificate := range searchresult.Results {
		if certificate.ParsedFingerprintSha256 == sha256fp {
			// The current cert is not a candidate
			continue
		}
		candidates = append(candidates, certificate.ParsedFingerprintSha256)
		if debug {
			fmt.Printf("Found candidate with fingerprint %s\n", certificate.ParsedFingerprintSha256)
		}
	}

	for _, candidate := range candidates {
		certdetails := get_certificate_by_sha256(client, username, password, candidate)

		if debug {
			fmt.Printf("Candidate %s was issued at %s\n", candidate, certdetails.Parsed.Validity.Start)
		}

		// Parsed.Validity.Start
		var isnewer bool
		isnewer = (certdetails.Parsed.Validity.Start).After(notbefore)

		if isnewer && replacement.Parsed.SerialNumber == "" {
			// This is the first possible replacement we found
			replacement = certdetails
		} else {
			if (certdetails.Parsed.Validity.Start).After(replacement.Parsed.Validity.Start) {
				// This is an even newer replacement
				replacement = certdetails
			}
		}

		if debug {
			fmt.Printf("Best candidate currently is %s\n", replacement.Parsed.FingerprintSha256)
		}
	}

	if replacement.Parsed.SerialNumber == "" {
		log.Fatal("No new certificate found\n");
		os.Exit(1)
	}

	// Download "parents" from "nss" and write fullchain, if enabled
	if fullchain != "" {
		fc, err := os.Create(fullchain)
		if err != nil {
			log.Fatal("Could not open fullchain file: %s\n", err)
			os.Exit(4)
		}
		defer fc.Close()

		// Full chain should include the new certificate
		_, _ = fc.WriteString(format_certificate(replacement.Raw, 64))

		// Download parent certificates and write them to full chain file
		for _, parent := range replacement.Validation.Nss.Parents {
			parentdetails := get_certificate_by_sha256(client, username, password, parent)
			_, _ = fc.WriteString(format_certificate(parentdetails.Raw, 64))
		}
		fc.Sync()
	}

	// Write out new certificate
	crt, err := os.Create(cert)
	if err != nil {
		log.Fatal("Could not open certificate file for writing: %s\n", err)
		os.Exit(4)
	}
	defer crt.Close()

	_, _ = crt.WriteString(format_certificate(replacement.Raw, 64))
	crt.Sync()

	os.Exit(0)
}

func get_cert_details (filename string) (string, time.Time, string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Could not read file: %s\n", filename)
		os.Exit(2)
	}

	block, _ := pem.Decode(file)
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatalf("Could not parse certificate: %s\n", err)
		os.Exit(2)
	}

	return (cert.Subject).String(), cert.NotBefore, fmt.Sprintf("%x", sha256.Sum256(cert.Raw))
}

func format_certificate (raw string, limit int) (string) {
	var result string

	result += "-----BEGIN CERTIFICATE-----\n"

	// https://www.socketloop.com/tutorials/golang-chunk-split-or-divide-a-string-into-smaller-chunk-example
	var charSlice []rune
	for _, char := range raw {
		charSlice = append(charSlice, char)
	}

	for len(charSlice) >= 1 {
		result += string(charSlice[:limit]) + "\n"
		charSlice = charSlice[limit:]

		if len(charSlice) < limit {
			limit = len(charSlice)
		}
	}

	result += "-----END CERTIFICATE-----\n"

	return result
}

func get_certificate_by_sha256 (client *http.Client, username string, password string, sha256 string) (CertificateDetails) {

	req, _ := http.NewRequest("GET", "https://censys.io/api/v1/view/certificates/" + sha256, nil)
	req.SetBasicAuth(username, password)
	resp, _ := client.Do(req)

	body, _ := ioutil.ReadAll(resp.Body)

	var certdetails CertificateDetails
	_ = json.Unmarshal(body, &certdetails)

	return certdetails
}
