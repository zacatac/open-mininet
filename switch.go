package mn

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

type Switch struct {
	Name       string
	Ports      Links
	Controller string
}

func (this Switch) String() string {
	out, err := json.MarshalIndent(this, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

func NewSwitch(name ...string) (*Switch, error) {
	this := &Switch{
		Name:  "",
		Ports: make(Links, 0),
	}

	if len(name) == 0 || name[0] == "" {
		this.Name = switchname()
	} else {
		this.Name = name[0]
	}

	if this.Exists() {
		return this, nil
	}

	if err := this.Create(); err != nil {
		return this, err
	}

	return this, nil
}

func (this *Switch) UnmarshalJSON(b []byte) error {
	type tmp Switch
	s := tmp{}

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	this.Name = s.Name
	this.Ports = s.Ports
	if !this.Exists() {
		if err := this.Create(); err != nil {
			return err
		}
	}

	if s.Controller != "" {
		if err := this.SetController(s.Controller); err != nil {
			return err
		}
	}

	return nil
}

func (this *Switch) Create() error {
	out, err := RunCommand("ovs-vsctl", "add-br", this.Name)
	if err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	return nil
}

func (this *Switch) Exists() bool {
	_, err := RunCommand("ovs-vsctl", "br-exists", this.Name)
	return err == nil
}

func (this *Switch) AddLink(l Link) error {
	if l.patch {
		return this.AddPatchPort(l)
	}

	return this.AddPort(l)
}

func (this *Switch) AddPort(l Link) error {
	out, err := RunCommand("ovs-vsctl", "add-port", this.Name, l.Name)
	if err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	this.Ports = append(this.Ports, l)

	return nil
}

func (this *Switch) AddPatchPort(l Link) error {
	if out, err := RunCommand("ovs-vsctl", "add-port", this.NodeName(), l.Name); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	if out, err := RunCommand("ovs-vsctl", "set", "interface", l.Name, "type=patch"); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	if out, err := RunCommand("ovs-vsctl", "set", "interface", l.Name, "options:peer="+l.Peer.Name); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	l = l.SetState("UP")

	this.Ports = append(this.Ports, l)

	return nil
}

func (this *Switch) SetController(addr string) error {
	if out, err := RunCommand("ovs-vsctl", "set-controller", this.NodeName(), addr); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	// if out, err := RunCommand("ovs-vsctl", "set", "bridge", this.NodeName(), "protocols=OpenFlow13"); err != nil {
	// 	return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	// }

	// if out, err := RunCommand("ovs-vsctl", "set", "bridge", this.NodeName()); err != nil {
	// 	return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	// }

	return nil
}

func (this Switch) Release() error {
	out, err := RunCommand("ovs-vsctl", "del-br", this.Name)
	if err != nil {
		log.Println("Unable to delete bridge", this.Name, err, out)
	}

	return nil
}

func (this Switch) NodeName() string {
	return this.Name
}

func (this Switch) NetNs() *NetNs {
	return nil
}

func (this Switch) LinksCount() int {
	return len(this.Ports)
}

func (this Switch) GetCidr(peer Peer) string {
	return this.Ports.LinkByPeer(peer).Cidr
}

func (this Switch) GetHwAddr(peer Peer) string {
	return this.Ports.LinkByPeer(peer).HwAddr
}

func (this Switch) GetState(peer Peer) string {
	return this.Ports.LinkByPeer(peer).State
}

func (this Switch) GetLinks() Links {
	return this.Ports
}
