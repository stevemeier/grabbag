package main

// Remove unused files from repodata directory
// Author: Steve Meier
// Date: 2020-03-12

import "encoding/xml"
import "io/ioutil"
import "log"
import "os"
import "path/filepath"

// from https://www.onlinetool.io/xmltogo/
type Repomd struct {
	XMLName  xml.Name `xml:"repomd"`
	Text     string   `xml:",chardata"`
	Xmlns    string   `xml:"xmlns,attr"`
	Rpm      string   `xml:"rpm,attr"`
	Revision string   `xml:"revision"`
	Data     []struct {
		Text     string `xml:",chardata"`
		Type     string `xml:"type,attr"`
		Checksum struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
		} `xml:"checksum"`
		OpenChecksum struct {
			Text string `xml:",chardata"`
			Type string `xml:"type,attr"`
		} `xml:"open-checksum"`
		Location struct {
			Text string `xml:",chardata"`
			Href string `xml:"href,attr"`
		} `xml:"location"`
		Timestamp       string `xml:"timestamp"`
		Size            string `xml:"size"`
		OpenSize        string `xml:"open-size"`
		DatabaseVersion string `xml:"database_version"`
	} `xml:"data"`
}

func main() {
	var err error
	var repofiles = make(map[string]bool)

	// Files which are always preserved
	repofiles["repomd.xml"] = true
	repofiles["repomd.xml.asc"] = true

	if len(os.Args) >= 2 {
		err := os.Chdir(os.Args[1])
		if err != nil { log.Fatalf("ERROR: %s\n", err.Error()) }
	}

	// Check if repomd.xml is there
	if !file_exists("repomd.xml") {
		log.Println("ERROR: repomd.xml not found")
		os.Exit(1)
	}

	// Open it
	file, err := os.Open("repomd.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read file names from it into map
	buf, err := ioutil.ReadFile("repomd.xml")
	var repomd Repomd
	err = xml.Unmarshal(buf, &repomd)
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	for _, data := range repomd.Data {
		repofiles[filepath.Base(data.Location.Href)] = true
	}

	// Read file names from directory and remove if not in map
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}
	for _, file := range files {
		if repofiles[file.Name()] == false {
			err := os.Remove(file.Name())
			if err != nil {
				log.Printf("Failed to remove %s: %s\n", file.Name(), err.Error())
				os.Exit(3)
			}
		}
	}
}

func file_exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
