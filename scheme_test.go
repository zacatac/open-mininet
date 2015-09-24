package mn

import (
	"testing"
	"time"
)

const exampleScheme = "apps/example.json"

func TestSchemeSimple(t *testing.T) {
	scheme := NewScheme()

	defer scheme.Release()

	s1, err := NewSwitch()
	if err != nil {
		t.Fatal(err)
	}

	h1, err := NewHost()
	if err != nil {
		t.Fatal(err)
	}

	scheme.AddNode(s1).AddNode(h1)

	n, found := scheme.GetNode(s1.NodeName())
	if !found {
		t.Fatal("Node", s1.NodeName(), "not found")
	}
	if n.NodeName() != s1.NodeName() {
		t.Fatal("\nExpected:", s1.NodeName(), "\n", "Obtained:", n.NodeName())
	}

	n, found = scheme.GetSwitch(s1.NodeName())
	if !found {
		t.Fatal("Switch", s1.NodeName(), "not found")
	}
	if n.NodeName() != s1.NodeName() {
		t.Fatal("\nExpected:", s1.NodeName(), "\n", "Obtained:", n.NodeName())
	}

	n, found = scheme.GetNode(h1.NodeName())
	if !found {
		t.Fatal("Node", h1.NodeName(), "not found")
	}
	if n.NodeName() != h1.NodeName() {
		t.Fatal("\nExpected:", h1.NodeName(), "\n", "Obtained:", n.NodeName())
	}

	n, found = scheme.GetHost(h1.NodeName())
	if !found {
		t.Fatal("Host", s1.NodeName(), "not found")
	}
	if n.NodeName() != h1.NodeName() {
		t.Fatal("\nExpected:", h1.NodeName(), "\n", "Obtained:", n.NodeName())
	}
}

// @todo make it "self-hosted". hardcoded to example.json
func TestSchemeFromJson(t *testing.T) {
	scheme, err := NewSchemeFromJson(exampleScheme)
	if err != nil {
		t.Fatal(err)
	}

	defer scheme.Release()

	if c := len(scheme.Switches); c != 2 {
		t.Fatal("Expected 2 switches, obtained:", c)
	}

	if c := len(scheme.Hosts); c != 3 {
		t.Fatal("Expected 3 switches, obtained:", c)
	}

	r1, found := scheme.GetNode("r1")
	if !found {
		t.Fatal("Expected node 'r1' not found")
	}

	if c := r1.LinksCount(); c != 2 {
		t.Fatal("Expected links count = 2, obtained:", c)
	}
}

// @todo test everything created
func TestSchemeApply(t *testing.T) {
	scheme, err := NewSchemeFromJson(exampleScheme)
	if err != nil {
		t.Fatal(err)
	}

	defer scheme.Release()

	s1, found := scheme.GetNode("s1")
	if !found {
		t.Fatal("Expected switch", s1.NodeName(), "not found")
	}

	if !s1.(*Switch).Exists() {
		t.Fatal("Expected switch", s1.NodeName(), "does not exist")
	}

	h1, found := scheme.GetNode("net1-h1")
	if !found {
		t.Fatal("Expected host", h1.NodeName(), "not found")
	}

	if !h1.(*Host).NetNs().Exists() {
		t.Fatal("Expected netns", h1.NetNs().Name(), "does not exist")
	}

	scheme.Recover()

	// wait until ping ends
	time.Sleep(time.Second * 5)
}
