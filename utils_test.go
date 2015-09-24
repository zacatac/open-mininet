package mn

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

func ifaceNotExists(ifname string, netns string) bool {
	var out string
	var err error

	if netns != "" {
		out, err = RunCommand("ip", "netns", "exec", netns, "ip", "link", "show", ifname)
	} else {
		out, err = RunCommand("ip", "link", "show", ifname)
	}

	if err != nil {
		log.Println(err, "output:", out)
		return true
	}

	if strings.Contains(out, "does not exist") {
		return true
	}

	return false
}

func ifaceUp(ifname string, netns string) bool {
	var out string
	var err error

	if netns != "" {
		out, err = RunCommand("ip", "netns", "exec", netns, "ip", "link", "show", ifname)
	} else {
		out, err = RunCommand("ip", "link", "show", ifname)
	}

	if err != nil {
		return false
	}

	if strings.Contains(out, "UP") {
		return true
	}

	return false
}

func ifaceAddr(ifname string, netns string, addr string) bool {
	var out string
	var err error

	if netns != "" {
		out, err = RunCommand("ip", "netns", "exec", netns, "ifconfig", ifname)
	} else {
		out, err = RunCommand("ifconfig", ifname)
	}

	if err != nil {
		log.Println(err, "output:", out)
		return false
	}

	if strings.Contains(out, addr) {
		return true
	}

	return false
}

func routeExists(routes []Route, netns string) bool {
	out, err := RunCommand("ip", "netns", "exec", netns, "route", "-n")
	if err != nil {
		log.Println(err)
		return false
	}

	for _, route := range routes {
		ip, _, _ := net.ParseCIDR(route.Dst)
		if !strings.Contains(out, ip.String()) || !strings.Contains(out, route.Gw) {
			return false
		}
	}

	return true
}

func ping(h1, h2 *Host) error {
	out, err := RunCommand("ip", "netns", "exec", h1.NetNs().Name(), "ping", "-c1", h2.Links.LinkByPeer(h1.Links[0].Peer).Ip())
	if err != nil {
		return err
	}

	if !strings.Contains(out, "1 packets transmitted, 1 received") {
		return errors.New(fmt.Sprintf("Unexpected ping result: %s", out))
	}

	out, err = RunCommand("ip", "netns", "exec", h2.NetNs().Name(), "ping", "-c1", h1.Links.LinkByPeer(h2.Links[0].Peer).Ip())
	if err != nil {
		return err
	}

	if !strings.Contains(out, "1 packets transmitted, 1 received") {
		return errors.New(fmt.Sprintf("Unexpected ping result: %s", out))
	}

	return nil
}
