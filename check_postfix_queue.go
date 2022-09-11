package main

import "fmt"
import "io/ioutil"
import "math"
import "os"
import "github.com/DavidGamba/go-getoptions"
import "github.com/olorin/nagiosplugin"

func main() {
	// Postfix default spool directory
	var spooldir string = "/var/spool/postfix"

	// Sub-directories of spool
	var subdirs []string = []string{"active","corrupt","deferred","hold","incoming","maildrop"}

	// Read command-line options
	var warn int
	var crit int
	opt := getoptions.New()
	opt.StringVar(&spooldir, "spool", spooldir, opt.Alias("s"))
	opt.IntVar(&warn, "warn", 1, opt.Alias("w"))
	opt.IntVar(&crit, "crit", 2, opt.Alias("c"))
	opt.Parse(os.Args[1:])
	if len(os.Args[1:]) == 0 {
        	fmt.Print(opt.Help())
        	os.Exit(1)
        }

	// Initialize Nagios module
	check := nagiosplugin.NewCheck()
	defer check.Finish()

	// Check for spooldir
	if !directory_exists(spooldir) {
		check.AddResult(nagiosplugin.UNKNOWN, "Directory "+spooldir+" not found")
		check.Finish()
	}

	filecount := make(map[string]int)
	for _, dir := range subdirs {
		fullpath := spooldir + "/" + dir
		if !directory_exists(fullpath) {
			check.AddResult(nagiosplugin.UNKNOWN, "Directory "+fullpath+" not found")
			check.Finish()
		}

		filecount[dir] = directory_filecount(fullpath)
		check.AddPerfDatum(dir, "", float64(filecount[dir]), 0.0, math.Inf(1), float64(warn), float64(crit))
	}

	if filecount["deferred"] >= crit {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("%d mails in deferred folder", filecount["deferred"]))
		check.Finish()
	}
	if filecount["deferred"] >= warn {
		check.AddResult(nagiosplugin.WARNING, fmt.Sprintf("%d mails in deferred folder", filecount["deferred"]))
		check.Finish()
	}
	if filecount["deferred"] == -1 {
		check.AddResult(nagiosplugin.UNKNOWN, "Could not access deferred folder")
		check.Finish()
	}

	check.AddResult(nagiosplugin.OK, "Mailqueue OK")
}

func directory_exists (dir string) (bool) {
	_, err := os.Stat(dir)
	return !os.IsNotExist(err)
}

func directory_filecount (dir string) (int) {
	var result int

	dircontent, err := ioutil.ReadDir(dir)
	if err != nil { return 0 }

	for _, object := range dircontent {
		if !object.IsDir() {
			result++
		} else {
			result += directory_filecount(dir+"/"+object.Name())
		}
	}

	return result
}
