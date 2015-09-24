package pool

import (
	"crypto/rand"
	"fmt"
	"net"
)

type pool struct {
	cache   map[string]*net.IPNet
	pool    map[string]bool
	created bool
	preset  bool
}

var instance pool

func ThePool(args ...interface{}) pool {
	if len(args) > 0 && !instance.created {
		ip, ipnet, err := net.ParseCIDR(args[0].(string))
		if err != nil {
			panic(err)
		}

		key := ip.Mask(ipnet.Mask)
		instance = newPool()
		instance.cache[key.String()] = ipnet
		instance.preset = true
	}

	if instance.created == false {
		instance = newPool()
	}

	return instance
}

func newPool() pool {
	return pool{
		cache:   make(map[string]*net.IPNet),
		pool:    make(map[string]bool),
		created: true,
	}
}

// @todo refactor it. make private next() method, which do all stuff
// and NextCidr and NextAddr wrappers

// @todo check for ip uniqueness

func (this pool) NextCidr(args ...interface{}) string {
	var cidr string

	if len(args) == 1 {
		cidr = args[0].(string)
	} else {
		for k, _ := range this.cache {
			ipnet := this.cache[k]
			cidr = ipnet.String()
			break
		}
	}

	ip, ipnet, err, key := this.get(cidr)
	if err != nil {
		panic(err)
	}

	inc := func(ip net.IP) {
		for j := len(ip) - 1; j >= 0; j-- {
			ip[j]++
			if ip[j] > 0 {
				break
			}
		}
	}

	inc(ip)
	if !ipnet.Contains(ip) {
		ip, ipnet, _ = net.ParseCIDR(ipnet.String())
	}

	ipnet.IP = ip

	if _, found := this.cache[key]; !found {
		this.cache[key] = &net.IPNet{}
	}

	this.cache[key] = ipnet

	return ipnet.String()
}

func (this pool) NextAddr(args ...interface{}) string {
	var addr, netmask string
	var ipnet *net.IPNet

	switch len(args) {
	case 2:
		addr = args[0].(string)
		netmask = args[1].(string)
		ipnet = &net.IPNet{
			IP:   net.ParseIP(addr),
			Mask: net.IPMask(net.ParseIP(netmask)),
		}

	case 0:
		for k, _ := range this.cache {
			ipnet = this.cache[k]
			break
		}
	default:
		return ""
	}

	ip, _, _ := net.ParseCIDR(this.NextCidr(ipnet.String()))

	return ip.String()
}

func (this pool) NextMac(mac string) string {
	hw, err := net.ParseMAC(mac)
	if err != nil {
		return ""
	}

	buf := make([]byte, 6)

	_, err = rand.Read(buf)
	if err != nil {
		return ""
	}

	buf[0] |= 2
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", hw[0], hw[1], hw[2], buf[3], buf[4], buf[5])
}

func (this pool) get(cidr string) (net.IP, *net.IPNet, error, string) {
	ip, ipnet, err := net.ParseCIDR(cidr)

	// This is because inventory nic's contains cidr as a an ip address and
	// we can not use it as a keys in cache map. So we have to convert them
	// to the mask. E.g. cidr 10.200.22.31/16 becomes 10.200.0.0, and we use
	// it as a key.
	ip = ip.Mask(ipnet.Mask)

	if c, found := this.cache[ip.String()]; found {
		return c.IP, c, nil, ip.String()
	}

	return ip, ipnet, err, ip.String()
}

func (this pool) Preset() bool {
	return this.preset
}
