package mn

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Procs []*Process

type Process struct {
	*os.Process
	Command string
	Args    []string
	attr    os.ProcAttr
	Output  string
}

func (this Procs) Add(p *Process) {
	this = append(this, p)
}

func (this Procs) GetByPid(pid int) *Process {
	for i, _ := range this {
		if this[i].Process == nil {
			continue
		}
		if this[i].Pid == pid {
			return this[i]
		}
	}

	return nil
}

func (this Process) GetPid() int {
	if this.Process != nil {
		return this.Pid
	}

	return 0
}

func (this Process) Stop() error {
	if this.Process == nil {
		return errors.New(fmt.Sprintf("No such process: ", this.Command, this.Args))
	}

	if err := this.Signal(os.Interrupt); err != nil {
		return err
	}

	return nil
}

// os.FindProcess() actually doesn't find it on posix systems, it just
// populates the struct and does SetFinalizer
// Idiomatic go way to find process by pid is:
//
// 	if e := syscall.Kill(pid, syscall.Signal(0)); e != nil {
// 		return nil, NewSyscallError("find process", e)
// 	}
//
// But we can't be sure, that found process actually the same
// so it looks, that finding it by name makes sence.
func (this Process) findProcessByName(netns string) (*os.Process, error) {
	out, err := RunCommand("ps", "-A", "-eo", "%p,%a")
	if err != nil {
		return nil, err
	}

	name := this.Command + " " + strings.Join(this.Args, " ")

	lines := strings.Split(out, "\n")

	for _, line := range lines {
		parts := strings.Split(line, ",")
		if len(parts) != 2 {
			continue
		}
		if parts[1] != name {
			continue
		}

		pid, err := strconv.Atoi(strings.Trim(parts[0], " "))
		if err != nil {
			return nil, err
		}

		// check that process is in the right netns
		if netns == netnsByPid(pid) {
			return os.FindProcess(pid)
		}
	}

	return nil, nil
}

// Beware, the old versions of ip utility don't support 'identify' command
func netnsByPid(pid int) string {
	out, err := RunCommand("ip", "netns", "identify", strconv.Itoa(pid))
	if err != nil {
		log.Println(err)
		return ""
	}

	return strings.Trim(out, "\n")
}
