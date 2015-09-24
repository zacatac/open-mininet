package netapps

import (
	"log"
	"net"

	"github.com/3d0c/ogo"
	"github.com/3d0c/ogo/protocol/ofp10"
)

type Host struct {
	mac  net.HardwareAddr
	port uint16
}

func NewL2Forwarder() interface{} {
	return &L2Forwarder{NewHostMap()}
}

type L2Forwarder struct {
	*hostmap
}

func (this *L2Forwarder) PacketIn(dpid net.HardwareAddr, pkt *ofp10.PacketIn) {
	eth := pkt.Data

	// Ignore link discovery packet types.
	if eth.Ethertype == 0xa0f1 || eth.Ethertype == 0x88cc {
		return
	}

	// this.SetHost(eth.HWSrc, pkt.InPort)
	this.hostmap.Add(eth.HWSrc, pkt.InPort)

	if host, ok := this.hostmap.Host(eth.HWDst); ok {
		if host.port == pkt.InPort {
			log.Println("Same port for packet from %s -> %s on %s.%s\n", eth.HWSrc, eth.HWDst, dpid, host.port)
			return
		}

		f1 := ofp10.NewFlowMod()

		f1.Match.DLSrc = eth.HWSrc
		f1.Match.DLDst = eth.HWDst
		f1.Match.InPort = pkt.InPort
		f1.Flags = ofp10.FC_ADD
		f1.Match.Wildcards = ofp10.FW_ALL ^ ofp10.FW_DL_SRC ^ ofp10.FW_DL_DST

		f1.AddAction(ofp10.NewActionOutput(host.port))
		f1.IdleTimeout = 10

		f2 := ofp10.NewFlowMod()
		f2.Match.DLSrc = eth.HWDst
		f2.Match.DLDst = eth.HWSrc
		f2.Match.InPort = host.port
		f2.AddAction(ofp10.NewActionOutput(pkt.InPort))
		f2.IdleTimeout = 3

		log.Println("Installing flow for", eth.HWSrc, pkt.InPort, "<-->", eth.HWDst, host.port)
		log.Println("Installing flow for", eth.HWDst, host.port, "<-->", eth.HWSrc, pkt.InPort)

		if s, ok := ogo.Switch(dpid); ok {
			s.Send(f1)
			s.Send(f2)
		}
	} else {
		p := ofp10.NewPacketOut()
		p.InPort = pkt.InPort
		p.BufferId = pkt.BufferId
		p.Data = &eth

		p.AddAction(ofp10.NewActionOutput(ofp10.P_FLOOD))

		if sw, ok := ogo.Switch(dpid); ok {
			sw.Send(p)
		}
	}
}
