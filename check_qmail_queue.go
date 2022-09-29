package main

import "fmt"
import "io/ioutil"
import "os"

import "github.com/DavidGamba/go-getoptions"
import "github.com/olorin/nagiosplugin"

func main() {
	var queuedir string = "/var/qmail/queue"

	// Read command-line options
        var warn int
        var crit int
	var countlocal bool
	var countremote bool
        opt := getoptions.New()
        opt.StringVar(&queuedir, "queue", queuedir, opt.Alias("d"))
	opt.BoolVar(&countlocal, "local", false)
	opt.BoolVar(&countremote, "remote", false)
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
        if !directory_exists(queuedir) {
                check.AddResult(nagiosplugin.UNKNOWN, "Directory "+queuedir+" not found")
                check.Finish()
        }

	if !countlocal &&
	   !countremote {
		check.AddResult(nagiosplugin.UNKNOWN, fmt.Sprint("Checked neither local nor remote queue"))
		check.Finish()
	}

	var filecount int
	if countlocal {
		filecount += directory_filecount(queuedir + "/local")
	}
	if countremote {
		filecount += directory_filecount(queuedir + "/remote")
	}

	if filecount >= crit {
		check.AddResult(nagiosplugin.CRITICAL, fmt.Sprintf("%d mails in queue", filecount))
		check.Finish()
	}
	if filecount >= warn {
		check.AddResult(nagiosplugin.WARNING, fmt.Sprintf("%d mails in queue", filecount))
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
