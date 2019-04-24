package main

import "encoding/xml"
//import "fmt"
import "github.com/davecgh/go-spew/spew"
import "io/ioutil"
import "os"
import "regexp"


type OvalData struct {
	Description	string
	References	[]string
	Rights		string
}

func main() {
	var oval map[string]OvalData
	oval = ParseOval("")
	spew.Dump(oval)
	oval = ParseOval("/Users/smeier/tmp/com.redhat.rhsa-all.xml2")
	spew.Dump(oval)
	oval = ParseOval("/Users/smeier/tmp/com.redhat.rhsa-all.xml")
	spew.Dump(oval)
}

func ParseOval(file string) map[string]OvalData {
	if file == "" {
		return nil
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil
	}

	// OvalDefinitions was generated 2019-04-24 22:06:30 by root on localhost.localdomain.
	type OvalDefinitions struct {
		XMLName        xml.Name `xml:"oval_definitions"`
		Text           string   `xml:",chardata"`
		Xmlns          string   `xml:"xmlns,attr"`
		Oval           string   `xml:"oval,attr"`
		RedDef         string   `xml:"red-def,attr"`
		UnixDef        string   `xml:"unix-def,attr"`
		Xsi            string   `xml:"xsi,attr"`
		SchemaLocation string   `xml:"schemaLocation,attr"`
		Generator      struct {
			Text           string `xml:",chardata"`
			ProductName    string `xml:"product_name"`
			ProductVersion string `xml:"product_version"`
			SchemaVersion  string `xml:"schema_version"`
			Timestamp      string `xml:"timestamp"`
			ContentVersion string `xml:"content_version"`
		} `xml:"generator"`
		Definitions struct {
			Text       string `xml:",chardata"`
			Definition []struct {
				Text     string `xml:",chardata"`
				Class    string `xml:"class,attr"`
				ID       string `xml:"id,attr"`
				Version  string `xml:"version,attr"`
				Metadata struct {
					Text     string `xml:",chardata"`
					Title    string `xml:"title"`
					Affected struct {
						Text     string   `xml:",chardata"`
						Family   string   `xml:"family,attr"`
						Platform []string `xml:"platform"`
					} `xml:"affected"`
					Reference []struct {
						Text   string `xml:",chardata"`
						RefID  string `xml:"ref_id,attr"`
						RefURL string `xml:"ref_url,attr"`
						Source string `xml:"source,attr"`
					} `xml:"reference"`
					Description string `xml:"description"`
					Advisory    struct {
						Text     string `xml:",chardata"`
						From     string `xml:"from,attr"`
						Severity string `xml:"severity"`
						Rights   string `xml:"rights"`
						Issued   struct {
							Text string `xml:",chardata"`
							Date string `xml:"date,attr"`
						} `xml:"issued"`
						Updated struct {
							Text string `xml:",chardata"`
							Date string `xml:"date,attr"`
						} `xml:"updated"`
						Cve []struct {
							Text   string `xml:",chardata"`
							Href   string `xml:"href,attr"`
							Public string `xml:"public,attr"`
							Impact string `xml:"impact,attr"`
							Cwe    string `xml:"cwe,attr"`
							Cvss2  string `xml:"cvss2,attr"`
							Cvss3  string `xml:"cvss3,attr"`
						} `xml:"cve"`
						Bugzilla []struct {
							Text string `xml:",chardata"`
							Href string `xml:"href,attr"`
							ID   string `xml:"id,attr"`
						} `xml:"bugzilla"`
						AffectedCpeList struct {
							Text string   `xml:",chardata"`
							Cpe  []string `xml:"cpe"`
						} `xml:"affected_cpe_list"`
					} `xml:"advisory"`
				} `xml:"metadata"`
				Criteria struct {
					Text      string `xml:",chardata"`
					Operator  string `xml:"operator,attr"`
					Criterion []struct {
						Text    string `xml:",chardata"`
						Comment string `xml:"comment,attr"`
						TestRef string `xml:"test_ref,attr"`
					} `xml:"criterion"`
					Criteria []struct {
						Text      string `xml:",chardata"`
						Operator  string `xml:"operator,attr"`
						Criterion []struct {
							Text    string `xml:",chardata"`
							Comment string `xml:"comment,attr"`
							TestRef string `xml:"test_ref,attr"`
						} `xml:"criterion"`
						Criteria []struct {
							Text     string `xml:",chardata"`
							Operator string `xml:"operator,attr"`
							Criteria []struct {
								Text      string `xml:",chardata"`
								Operator  string `xml:"operator,attr"`
								Criterion []struct {
									Text    string `xml:",chardata"`
									Comment string `xml:"comment,attr"`
									TestRef string `xml:"test_ref,attr"`
								} `xml:"criterion"`
							} `xml:"criteria"`
							Criterion []struct {
								Text    string `xml:",chardata"`
								Comment string `xml:"comment,attr"`
								TestRef string `xml:"test_ref,attr"`
							} `xml:"criterion"`
						} `xml:"criteria"`
					} `xml:"criteria"`
				} `xml:"criteria"`
			} `xml:"definition"`
		} `xml:"definitions"`
		Tests struct {
			Text        string `xml:",chardata"`
			RpminfoTest []struct {
				Text    string `xml:",chardata"`
				Check   string `xml:"check,attr"`
				Comment string `xml:"comment,attr"`
				ID      string `xml:"id,attr"`
				Version string `xml:"version,attr"`
				Object  struct {
					Text      string `xml:",chardata"`
					ObjectRef string `xml:"object_ref,attr"`
				} `xml:"object"`
				State struct {
					Text     string `xml:",chardata"`
					StateRef string `xml:"state_ref,attr"`
				} `xml:"state"`
			} `xml:"rpminfo_test"`
		} `xml:"tests"`
		Objects struct {
			Text          string `xml:",chardata"`
			RpminfoObject []struct {
				Text    string `xml:",chardata"`
				ID      string `xml:"id,attr"`
				Version string `xml:"version,attr"`
				Name    string `xml:"name"`
			} `xml:"rpminfo_object"`
		} `xml:"objects"`
		States struct {
			Text         string `xml:",chardata"`
			RpminfoState []struct {
				Text           string `xml:",chardata"`
				ID             string `xml:"id,attr"`
				AttrVersion    string `xml:"version,attr"`
				SignatureKeyid struct {
					Text      string `xml:",chardata"`
					Operation string `xml:"operation,attr"`
				} `xml:"signature_keyid"`
				Version struct {
					Text      string `xml:",chardata"`
					Operation string `xml:"operation,attr"`
				} `xml:"version"`
				Arch struct {
					Text      string `xml:",chardata"`
					Datatype  string `xml:"datatype,attr"`
					Operation string `xml:"operation,attr"`
				} `xml:"arch"`
				Evr struct {
					Text      string `xml:",chardata"`
					Datatype  string `xml:"datatype,attr"`
					Operation string `xml:"operation,attr"`
				} `xml:"evr"`
			} `xml:"rpminfo_state"`
		} `xml:"states"`
	}

	var ovaldata OvalDefinitions
	data, _ := ioutil.ReadFile(file)
        _ = xml.Unmarshal([]byte(data), &ovaldata)
	oval := make(map[string]OvalData)

	for _, def := range ovaldata.Definitions.Definition {
		id := def.ID
		id = "CESA-" + id[len(id)-8:len(id)-4] + ":" + id[len(id)-4:]

		var cves []string
		for _, ref := range def.Metadata.Reference {
			matched, _ := regexp.MatchString(`^CVE`, ref.RefID)
			if matched {
				cves = append(cves, ref.RefID)
			}
		}

		var current = oval[id]
		current.Description = def.Metadata.Description
		current.Rights = def.Metadata.Advisory.Rights
		current.References = cves
		oval[id] = current
	}

	return oval
}
