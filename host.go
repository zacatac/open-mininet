package mn

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

type Host struct {
	Cgroup *Cgroup
	Name   string
	netns  *NetNs
	Links  Links
	Procs  Procs
}

func (this Host) String() string {
	out, err := json.MarshalIndent(this, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

func NewRouter(name ...string) (*Host, error) {
	h, err := NewHost(name...)
	if err != nil {
		return nil, err
	}

	if err = h.EnableForwarding(); err != nil {
		return nil, err
	}

	return h, nil
}

func NewHost(name ...string) (*Host, error) {
	this := &Host{
		Name:  "",
		Links: make(Links, 0),
	}

	if len(name) == 0 || name[0] == "" {
		this.Name = hostname(1024)
	} else {
		this.Name = name[0]
	}

	var err error

	if this.netns, err = NewNetNs(this.Name); err != nil {
		return nil, err
	}

	return this, nil
}

func (this *Host) UnmarshalJSON(b []byte) error {
	type tmp Host
	host := tmp{}

	if err := json.Unmarshal(b, &host); err != nil {
		return err
	}

	this.Name = host.Name
	this.Links = host.Links
	this.netns = &NetNs{name: host.Name}
	this.Procs = host.Procs
	this.Cgroup = host.Cgroup

	if !this.netns.Exists() {
		if err := this.NetNs().Create(); err != nil {
			return err
		}
	}

	if len(this.Links) > 1 {
		this.EnableForwarding()
	}

	return nil
}

func (this *Host) RunProcess(args ...string) (*Process, error) {
	p, err := this.runProcess(args...)
	if err != nil {
		return nil, err
	}

	this.Procs = append(this.Procs, p)
	return p, nil
}

func (this *Host) runProcess(args ...string) (*Process, error) {
	var command []string

	if this.Cgroup != nil {
		command = this.Cgroup.CgExecCommand()
	}

	ipCmd := FullPathFor("ip")
	if ipCmd == "" {
		return nil, errors.New("ip command not found the PATH")
	}

	if this.NetNs() != nil {
		command = append(command, []string{ipCmd, "netns", "exec", this.NetNs().Name()}...)
	}

	command = append(command, args...)

	p := &Process{Command: args[0], Args: args[1:]}

	fname := fmt.Sprintf("/tmp/output.%d", time.Now().Nanosecond())
	pout, err := os.Create(fname)
	if err != nil {
		log.Println("Unable to create temp file", fname, "for process stderr/stdout")
		p.attr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	}

	// var procAttr os.ProcAttr
	// procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}
	p.attr.Files = []*os.File{nil, pout, pout}

	p.Output = fname

	process, err := os.StartProcess(command[0], command, &p.attr)
	if err != nil {
		return nil, err
	}

	// detach process
	// process.Release()

	p.Process = process

	fmt.Println("Started", command, "All output goes to", fname)

	go func() {
		pid := p.Pid
		s, err := process.Wait()
		if err != nil {
			panic(err)
		}

		for i, _ := range this.Procs {
			if this.Procs[i].Process == nil {
				continue
			}
			if this.Procs[i].Pid == pid {
				this.Procs[i].Pid = 0
			}
		}

		log.Printf("Process [%d] %v finished with %v, %v", pid, command, s.Exited(), s.String())
	}()

	return p, nil
}

func (this Host) RunCommand(args ...string) (string, error) {
	var command []string

	if this.Cgroup != nil {
		command = this.Cgroup.CgExecCommand()
	}

	ipCmd := FullPathFor("ip")
	if ipCmd == "" {
		return "", errors.New("ip command not found the PATH")
	}

	if this.NetNs() != nil {
		command = append(command, []string{ipCmd, "netns", "exec", this.NetNs().Name()}...)
	}

	command = append(command, args...)

	return RunCommand(command[0], command[1:]...)
}

func (this Host) EnableForwarding() error {
	_, err := this.RunCommand("sysctl", "net.ipv4.ip_forward=1")
	return err
}

func (this Host) NodeName() string {
	return this.Name
}

func (this Host) NetNs() *NetNs {
	return this.netns
}

func (this Host) LinksCount() int {
	return len(this.Links)
}

func (this Host) GetCidr(peer Peer) string {
	return this.Links.LinkByPeer(peer).Cidr
}

func (this Host) GetHwAddr(peer Peer) string {
	return this.Links.LinkByPeer(peer).HwAddr
}

func (this Host) GetState(peer Peer) string {
	return this.Links.LinkByPeer(peer).State
}

func (this Host) GetLinks() Links {
	return this.Links
}

func (this Host) Release() error {
	if err := this.netns.Release(); err != nil {
		log.Println(err)
	}

	for _, link := range this.Links {
		link.Release()
	}

	for _, proc := range this.Procs {
		proc.Stop()
	}

	this.Cgroup.Release()

	return nil
}

func (this *Host) AddLink(l Link) error {
	this.Links = append(this.Links, l)
	return nil
}

func (this *Host) recoverProcs() error {
	for i, proc := range this.Procs {
		fmt.Println("Recovering ", proc.Command, proc.Args)

		var p *os.Process
		var err error

		if p, err = proc.findProcessByName(this.NetNs().Name()); err != nil {
			return err
		}

		if p != nil {
			proc.Process = p
		} else {
			c := append([]string{proc.Command}, proc.Args...)
			result, err := this.runProcess(c...)
			if err != nil {
				return err
			}

			this.Procs[i].Process = result.Process

			continue
		}
	}

	return nil
}
