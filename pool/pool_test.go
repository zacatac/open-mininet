package pool

import (
	"testing"
)

func TestIPs(t *testing.T) {
	ip2 := ThePool().NextCidr("62.76.47.10/28")
	ip3 := ThePool().NextCidr("62.76.47.10/28")

	if ip2 != "62.76.47.1/28" {
		t.Fatal("Expected ip2 = 62.76.47.1/24, obtained =", ip2)
	}

	if ip3 != "62.76.47.2/28" {
		t.Fatal("Expected ip2 = 62.76.47.2/24, obtained =", ip3)
	}

}

func TestNextAddr(t *testing.T) {
	ip1 := ThePool().NextAddr("10.200.2.115", "255.255.0.0")
	ip2 := ThePool().NextAddr("10.200.2.115", "255.255.0.0")

	if ip1 != "10.200.0.1" {
		t.Fatal("Expected ip1 = 10.200.0.1, obtained =", ip1)
	}

	if ip2 != "10.200.0.2" {
		t.Fatal("Expected ip1 = 10.200.0.2, obtained =", ip2)
	}
}

func TestCustomRange(t *testing.T) {
	// init pool with custom range
	ThePool("192.168.0.0/24")

	ip1 := ThePool().NextAddr()
	ip2 := ThePool().NextAddr()

	if ip1 != "192.168.0.1" {
		t.Fatal("Expected ip1 = 192.168.0.2, obtained =", ip1)
	}

	if ip2 != "192.168.0.2" {
		t.Fatal("Expected ip2 = 192.168.0.3, obtained =", ip2)
	}

	ip3 := ThePool().NextCidr()

	if ip3 != "192.168.0.3/24" {
		t.Fatal("Expected ip3 = 192.168.0.3/24, obtained =", ip3)
	}
}
