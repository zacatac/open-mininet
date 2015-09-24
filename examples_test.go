package mn

import (
	"strings"
	"testing"

	"github.com/NodePrime/open-mininet/pool"
)

func TestTopoSimple(t *testing.T) {
	root, err := NewSwitch()
	if err != nil {
		t.Fatal(err)
	}

	defer root.Release()

	hosts := []*Host{}

	pool.ThePool("192.168.55.1/24")

	for i := 0; i < 10; i++ {
		host, err := NewHost(hostname(65535))
		if err != nil {
			t.Fatal(err)
		}

		defer host.Release()

		pair := NewLink(root, host, Link{Cidr: noip})
		pair.Create()

		if err := root.AddLink(pair.Left); err != nil {
			t.Fatal(err)
		}

		if err := pair.Left.Up(); err != nil {
			t.Fatal(err)
		}

		if err := pair.Right.ApplyCidr(); err != nil {
			t.Fatal(err)
		}

		if err := pair.Right.Up(); err != nil {
			t.Fatal(err)
		}

		host.AddLink(pair.Right)

		hosts = append(hosts, host)
	}

	for _, src := range hosts {
		for _, dst := range hosts {
			if src.NodeName() == dst.NodeName() {
				continue
			}

			out, err := RunCommand("ip", "netns", "exec", src.NodeName(), "ping", "-c1", dst.Links[0].Ip())
			if err != nil {
				t.Fatal(err, out)
			}

			if !strings.Contains(out, "1 received") {
				t.Fatal("Can't reach 192.168.55.1 from", dst.Links[0].Ip(), "\n",
					"Output:", out)
			}

		}
	}

}
