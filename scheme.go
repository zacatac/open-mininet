package mn

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
)

type Scheme struct {
	Switches []*Switch
	Hosts    []*Host
	pairs    map[string]bool
}

func (this Scheme) String() string {
	out, err := json.MarshalIndent(this, "", "      ")
	if err != nil {
		panic(err)
	}

	return string(out)
}

func NewScheme() *Scheme {
	return &Scheme{
		make([]*Switch, 0),
		make([]*Host, 0),
		make(map[string]bool),
	}
}

func NewSchemeFromJson(fname string) (*Scheme, error) {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		return nil, err
	}

	scheme := NewScheme()

	err = json.Unmarshal(data, scheme)
	if err != nil {
		return nil, err
	}

	return scheme, nil
}

func (this *Scheme) AddNode(n interface{}) *Scheme {
	switch t := n.(type) {
	case *Switch:
		this.Switches = append(this.Switches, n.(*Switch))
	case *Host:
		this.Hosts = append(this.Hosts, n.(*Host))
	default:
		log.Println("Wrong call, unknown type", t, "for", n)
	}

	return this
}

func (this *Scheme) GetNode(name string) (Node, bool) {
	if n, found := this.GetHost(name); found {
		return n, found
	}

	if n, found := this.GetSwitch(name); found {
		return n, found
	}

	return nil, false
}

func (this *Scheme) GetHost(name string) (*Host, bool) {
	for _, host := range this.Hosts {
		if host.NodeName() == name {
			return host, true
		}
	}

	return nil, false
}

func (this *Scheme) GetSwitch(name string) (*Switch, bool) {
	for _, sw := range this.Switches {
		if sw.NodeName() == name {
			return sw, true
		}
	}

	return nil, false
}

func (this *Scheme) Nodes() chan Node {
	yield := make(chan Node)

	go func() {
		for _, sw := range this.Switches {
			yield <- (Node)(sw)
		}

		for _, host := range this.Hosts {
			yield <- (Node)(host)
		}

		close(yield)
	}()

	return yield

}

func (this Scheme) Export() string {
	return this.String()
}

func (this Scheme) Recover() error {

	for node := range this.Nodes() {
		switch t := node.(type) {
		case *Switch:
			this.recoverSwitchPorts(node.(*Switch))
		case *Host:
			this.recoverHostLinks(node.(*Host))
		default:
			log.Println("Unexpected type", t)
		}
	}

	for _, host := range this.Hosts {
		if err := host.recoverProcs(); err != nil {
			return err
		}
	}

	return nil
}

// Recover switch to host connectivity
func (this Scheme) recoverSwitchPorts(s *Switch) error {
	for _, port := range s.Ports {
		if port.Exists() {
			continue
		}

		peer, found := this.GetNode(port.Peer.NodeName)
		if !found {
			return errors.New(fmt.Sprintf("Can't find host %s", port.Peer.NodeName))
		}

		link := peer.GetLinks().LinkByPeer(port.Peer)
		pair := Pair{port, link}

		hash := port.Name + "-" + link.Name
		if this.pairs[hash] {
			log.Println("Wrong scheme. Two identic pairs found:", pair, "Skipping.")
			continue
		}

		// patch link
		if s2, found := this.GetSwitch(peer.NodeName()); found {
			s.AddPatchPort(pair.Left)
			s2.AddPatchPort(pair.Right)
			continue
		}

		if err := pair.Create(); err != nil {
			return err
		}

		if err := s.AddLink(pair.Left); err != nil {
			return err
		}

		_, err := pair.Up()
		if err != nil {
			return err
		}

		this.pairs[hash] = true
	}

	return nil
}

// Recover host to host connectivity  @todo
func (this Scheme) recoverHostLinks(h *Host) error {
	for _, left := range h.Links {
		peer, found := this.GetHost(left.Peer.NodeName)
		if !found {
			continue
		}

		right := peer.Links.LinkByPeer(left.Peer)
		if right.NodeName == "" {
			// nothing found
			// @todo-maybe return (Link, bool) form LinkByPeer
			continue
		}

		pair := Pair{left, right}

		hash := left.NodeName + left.Name + right.NodeName + right.Name
		if this.pairs[hash] {
			continue
		}

		if err := pair.Create(); err != nil {
			return err
		}

		_, err := pair.Up()
		if err != nil {
			return err
		}

		h.AddLink(left)

		h2, found := this.GetHost(right.NodeName)
		if !found {
			return errors.New(fmt.Sprintf("Can't find host node", right.NodeName))
		}

		h2.AddLink(right)

		this.pairs[hash] = true
	}

	return nil
}

func (this *Scheme) Release() {
	for node := range this.Nodes() {
		node.Release()
	}
}
