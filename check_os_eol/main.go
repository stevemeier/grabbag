package main

import "fmt"
import "os"
import "regexp"
import "time"
import "github.com/acobaugh/osrelease"
import "github.com/DavidGamba/go-getoptions"
import "github.com/olorin/nagiosplugin"

type EOLdata struct {
	ID		string
	VersionID	string
	Ultimo		time.Time
}

var eoldata []EOLdata

func load_eoldata () {
	// EOL data mostly from https://endoflife.date/
	// Raspbian does not seem to an offical page/document as of Dec 26, 2022 (debian assumed)

	// Synology is well hidden at
	// https://global.download.synology.com/download/Document/Software/WhitePaper/Firmware/DSM/All/enu/Synology_Security_Whitepaper.pdf
	eoldata = []EOLdata{
	EOLdata{ID: "centos", VersionID: "6", Ultimo: time.Date(2020, 11, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "centos", VersionID: "7", Ultimo: time.Date(2024,  6, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "centos", VersionID: "8", Ultimo: time.Date(2021, 12, 31, 0,0,0,0,time.UTC)},
	EOLdata{ID: "centos", VersionID: "9", Ultimo: time.Date(2027,  5, 31, 0,0,0,0,time.UTC)},

	EOLdata{ID: "debian", VersionID:  "8", Ultimo: time.Date(2020,  6, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "debian", VersionID:  "9", Ultimo: time.Date(2022,  6, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "debian", VersionID: "10", Ultimo: time.Date(2024,  6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "debian", VersionID: "11", Ultimo: time.Date(2026,  8, 15, 0,0,0,0,time.UTC)},

	EOLdata{ID: "fedora", VersionID: "30", Ultimo: time.Date(2020,  6, 26, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "31", Ultimo: time.Date(2020, 11, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "32", Ultimo: time.Date(2021,  5, 25, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "33", Ultimo: time.Date(2021, 11, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "34", Ultimo: time.Date(2022,  6,  7, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "35", Ultimo: time.Date(2022, 12, 13, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "36", Ultimo: time.Date(2023,  5, 16, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "37", Ultimo: time.Date(2023, 12, 15, 0,0,0,0,time.UTC)},
	EOLdata{ID: "fedora", VersionID: "38", Ultimo: time.Date(2024,  5, 14, 0,0,0,0,time.UTC)},

	EOLdata{ID: "raspbian", VersionID:  "9", Ultimo: time.Date(2022, 6, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "raspbian", VersionID: "10", Ultimo: time.Date(2024, 6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "raspbian", VersionID: "11", Ultimo: time.Date(2026, 8, 15, 0,0,0,0,time.UTC)},

	EOLdata{ID: "rhel", VersionID: "6", Ultimo: time.Date(2022, 11, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "rhel", VersionID: "7", Ultimo: time.Date(2024,  6, 30, 0,0,0,0,time.UTC)},
	EOLdata{ID: "rhel", VersionID: "8", Ultimo: time.Date(2029,  5, 31, 0,0,0,0,time.UTC)},
	EOLdata{ID: "rhel", VersionID: "9", Ultimo: time.Date(2032,  5, 31, 0,0,0,0,time.UTC)},

	EOLdata{ID: "synology_dsm", VersionID: "6.0", Ultimo: time.Date(2018,  6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "synology_dsm", VersionID: "6.1", Ultimo: time.Date(2019,  6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "synology_dsm", VersionID: "6.2", Ultimo: time.Date(2021,  6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "synology_dsm", VersionID: "7.0", Ultimo: time.Date(2023,  6,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "synology_dsm", VersionID: "7.1", Ultimo: time.Date(2024,  6,  1, 0,0,0,0,time.UTC)},

	EOLdata{ID: "ubuntu", VersionID: "14.04", Ultimo: time.Date(2024, 4,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "16.04", Ultimo: time.Date(2026, 4,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "18.04", Ultimo: time.Date(2028, 4,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "19.10", Ultimo: time.Date(2020, 7,  6, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "20.04", Ultimo: time.Date(2030, 4,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "20.10", Ultimo: time.Date(2021, 7, 22, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "21.04", Ultimo: time.Date(2022, 1, 20, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "21.10", Ultimo: time.Date(2022, 7, 14, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "22.04", Ultimo: time.Date(2032, 4,  1, 0,0,0,0,time.UTC)},
	EOLdata{ID: "ubuntu", VersionID: "22.10", Ultimo: time.Date(2023, 7, 20, 0,0,0,0,time.UTC)},
	}
}

func main() {
	load_eoldata()

	var warn int
	var list bool
	opt := getoptions.New()
	opt.IntVar(&warn, "warn", 180, opt.Alias("w"))
	opt.BoolVar(&list, "list", false, opt.Alias("l"))
	opt.Parse(os.Args[1:])

	if list {
		for _, eolobj := range eoldata {
			fmt.Printf("ID: %s | Version: %s | EOL: %s\n", eolobj.ID, eolobj.VersionID, eolobj.Ultimo.Format(time.RFC3339))
		}

		os.Exit(0)
	}

	// Initialize Nagios module
	check := nagiosplugin.NewCheck()
	defer check.Finish()

	osrel, initerr := osrelease.Read()

	if initerr != nil {
		// Synology is stupid
		_, staterr := os.Stat("/etc/VERSION")
		if staterr == nil {
			osrel, initerr = osrelease.ReadFile("/etc/VERSION")
			if initerr == nil {
				osrel["ID"] = "synology_dsm"
				osrel["VERSION_ID"] = fmt.Sprintf("%s.%s", osrel["majorversion"], osrel["minorversion"])
			}
		}
	}

	if initerr != nil {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("Failed to init os-release: %s", initerr.Error()))
		check.Finish()
	}

	// Alma and Rocky seem to have the same EOL policy as Red Hat, so we map them
	if osrel["ID"] == "almalinux" || osrel["ID"] == "rocky" {
		osrel["ID"] = "rhel"
	}

	var osfound bool
	var iseol bool
	var eoloffset time.Duration
	osfound, iseol, eoloffset = is_eol(osrel["ID"], osrel["VERSION_ID"])
	if !osfound {
		// For distributions like Red Hat, we need the major version only (8, instead of 8.8)
		osfound, iseol, eoloffset = is_eol(osrel["ID"], extract_major(osrel["VERSION_ID"]))
	}

	if iseol {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("OS has reached EOL %d days ago", days(eoloffset)))
		check.Finish()
	}

	if !osfound {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprintf("No EOL information available for this OS: %s / %s", osrel["ID"], osrel["VERSION_ID"]))
		check.Finish()
	}

	if days(eoloffset) > (warn * -1) {
		check.AddResult(nagiosplugin.WARNING, fmt.Sprintf("OS will reach EOL in %d days", days(eoloffset) * -1))
		check.Finish()
	}

	check.AddResult(nagiosplugin.OK, fmt.Sprintf("OS is still supported (%d days remaining)", days(eoloffset) * -1))
}

func is_eol (id string, version string) (bool, bool, time.Duration) {
	now := time.Now()
	for _, eolobj := range eoldata {
		if id == eolobj.ID && version == eolobj.VersionID {
			return true, now.After(eolobj.Ultimo), now.Sub(eolobj.Ultimo)
		}
	}
		
	return false, false, time.Duration(0 * time.Second)
}

func days (t time.Duration) (int) {
	return int(t.Hours() / 24)
}

func extract_major (id string) (string) {
	re := regexp.MustCompile("^\\d+")
	matches := re.FindStringSubmatch(id)
	if len(matches) > 0 {
		return matches[0]
	}
	return "x"
}
