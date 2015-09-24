package main

import (
	mn "github.com/NodePrime/open-mininet"
	"time"
)

func main() {
	// creating and instance of Scheme struct, which is just a
	// a storage for futher nodes
	scheme := mn.NewScheme()

	defer scheme.Release()

	// create new host h1
	host1, err := mn.NewHost("h1")
	if err != nil {
		panic(err)
	}

	// create new switch with random name
	sw, err := mn.NewSwitch()
	if err != nil {
		panic(err)
	}

	// interconnect nodes
	pair := mn.NewLink(sw, host1, mn.Link{Cidr: "noip"}, mn.Link{Cidr: "192.168.44.1/24"})

	// physically create link
	if err := pair.Create(); err != nil {
		panic(err)
	}

	// apply cidr, routes, bring interfaces up
	pair, err = pair.Up()
	if err != nil {
		panic(err)
	}

	sw.AddLink(pair.Left)
	host1.AddLink(pair.Right)

	scheme.AddNode(host1)
	scheme.AddNode(sw)

	// repeat for host2

	host2, err := mn.NewHost("h2")
	if err != nil {
		panic(err)
	}

	pair2 := mn.NewLink(sw, host2, mn.Link{Cidr: "noip"}, mn.Link{Cidr: "192.168.44.2/24"})
	if err := pair2.Create(); err != nil {
		panic(err)
	}
	pair2, err = pair2.Up()
	if err != nil {
		panic(err)
	}

	sw.AddLink(pair2.Left)
	host2.AddLink(pair2.Right)

	scheme.AddNode(host2)

	// now we can run some command

	host1.RunProcess("ping", "-c1", "192.168.44.2")

	time.Sleep(time.Second * 1)
}
