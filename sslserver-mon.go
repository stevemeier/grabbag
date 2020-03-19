package main

import "bytes"
import "errors"
import "fmt"
import "os"
import "os/exec"
import "syscall"
import "time"
import "github.com/prometheus/procfs"

func main() {
	fs, _ := procfs.NewFS("/proc")
	allprocs, _ := fs.AllProcs()
	for _, proc := range allprocs {
		exe, _ := proc.Executable()
		if exe == "/usr/bin/sslserver" {
			stat, _ := proc.Stat()
			if stat.PPID == 1 {
				continue
			}
			starttime, _ := stat.StartTime()
			runtime := time.Now().Unix() - int64(starttime)
			fmt.Printf("[%d@%d] Runtime is %d seconds\n", stat.PID, time.Now().Unix(), runtime)
			fmt.Printf("[%d@%d] Usertime is %d\n", stat.PID, time.Now().Unix(), stat.UTime)
			fmt.Printf("[%d@%d] Kerneltime is %d\n", stat.PID, time.Now().Unix(), stat.STime)
			if runtime > 300 && stat.STime > 15000 {
				var output []byte
				fmt.Printf("Lsof for PID %d\n", stat.PID)
				lsofpid := fmt.Sprintf("%d", stat.PID)
				output, _, _ = sysexec("/sbin/lsof", []string{"-p", lsofpid}, nil)
				fmt.Print(string(output))

				fmt.Printf("Lsof for Parent %d\n", int(stat.PID) - 1)
				lsofppid := fmt.Sprintf("%d", (stat.PID - 1))
				output, _, _ = sysexec("/sbin/lsof", []string{"-p", lsofppid}, nil)
				fmt.Print(string(output))

				// Give some time before killing
				time.Sleep(3 * time.Second)
				fmt.Printf("Killing PID %d after %d runtime\n", stat.PID, runtime)
				syscall.Kill(stat.PID, 15)
			}
		}
	}
}

func sysexec (command string, args []string, input []byte) ([]byte, int, error) {
        var output bytes.Buffer

        if !file_exists(command) {
                return nil, 111, errors.New("command not found")
        }

        if !is_executable(command) {
                return nil, 111, errors.New("command not executable")
        }

        cmd := exec.Command(command, args...)
        cmd.Stdin = bytes.NewBuffer(input)
        cmd.Stdout = &output
        err := cmd.Run()

        exitcode := 0
        if exitError, ok := err.(*exec.ExitError); ok {
                exitcode = exitError.ExitCode()
        }

        return output.Bytes(), exitcode, err
}

func is_executable (file string) bool {
        stat, err := os.Stat(file)
        if err != nil {
                return false
        }

        // These calls return uint32 by default while
        // os.Get?id returns int. So we have to change one
        fileuid := int(stat.Sys().(*syscall.Stat_t).Uid)
        filegid := int(stat.Sys().(*syscall.Stat_t).Gid)

        if (os.Getuid() == fileuid) { return stat.Mode()&0100 != 0 }
        if (os.Getgid() == filegid) { return stat.Mode()&0010 != 0 }
        return stat.Mode()&0001 != 0
}

func file_exists(filename string) bool {
        info, err := os.Stat(filename)
        if os.IsNotExist(err) {
                return false
        }

        return !info.IsDir()
}
