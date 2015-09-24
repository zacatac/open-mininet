package netapps

import (
	"net"
	"sync"
)

type host struct {
	mac  net.HardwareAddr
	port uint16
}

type hostmap struct {
	byMacIp map[string]map[string]host
	byMac   map[string]host
	sync.RWMutex
}

func NewHostMap() *hostmap {
	return &hostmap{
		byMacIp: make(map[string]map[string]host),
		byMac:   make(map[string]host),
	}
}

func (this *hostmap) Host(v ...interface{}) (h host, ok bool) {
	this.RLock()
	defer this.RUnlock()

	switch len(v) {
	case 0:
		panic("Wrong call, expected at least one argument to hostMap.Host(...)")

	case 1:
		mac, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected net.HardwareAddr")
		}

		h, ok := this.byMac[mac.String()]
		return h, ok

	case 2:
		dpid, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("First argument expected to be net.HardwareAddr")
		}

		ip, ok := v[1].(net.IP)
		if !ok {
			panic("Second argument expected to be net.IP")
		}

		h, ok := this.byMacIp[dpid.String()][ip.String()]
		return h, ok
	}

	return
}

func (this *hostmap) Add(v ...interface{}) {
	this.RLock()
	defer this.RUnlock()

	switch len(v) {
	case 0:
		panic("Wrong call, expected at least one argument to hostMap.Host(...)")

	case 2:
		mac, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		port, ok := v[1].(uint16)
		if !ok {
			panic("Expected second argument to be an uint16")
		}

		this.byMac[mac.String()] = host{mac, port}
		return

	case 3:
		dpid, ok := v[0].(net.HardwareAddr)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		ip, ok := v[1].(net.IP)
		if !ok {
			panic("Expected first argument to be a net.HardwareAddr")
		}

		h, ok := v[2].(host)
		if !ok {
			panic("Expected third argument to be a host")
		}

		if _, found := this.byMacIp[dpid.String()]; !found {
			this.byMacIp[dpid.String()] = make(map[string]host)
		}

		this.byMacIp[dpid.String()][ip.String()] = h

		return

	default:
		panic("Wrong call, expected 2 or three arguments")
	}

}

func (this *hostmap) Dpid(dpid net.HardwareAddr) (map[string]host, bool) {
	iptohost, found := this.byMacIp[dpid.String()]
	return iptohost, found
}
