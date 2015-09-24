package netapps

import (
	"log"
	"net"

	"github.com/3d0c/ogo"
	"github.com/3d0c/ogo/protocol/arp"
	"github.com/3d0c/ogo/protocol/eth"
	"github.com/3d0c/ogo/protocol/ipv4"
	"github.com/3d0c/ogo/protocol/ofp10"
)

func NewL3Forwarder(f []string) func() interface{} {
	return func() interface{} {
		return &L3Forwarder{
			arpTable: NewHostMap(),
			fakeways: f,
		}
	}
}

type L3Forwarder struct {
	arpTable *hostmap
	fakeways []string
}

type pair struct {
	dpid   string
	ipaddr string
}

type buffer struct {
	bufferId uint32
	inport   uint16
}

var lostBuffers map[pair][]buffer = make(map[pair][]buffer)

func dpidToMac(dpid net.HardwareAddr) net.HardwareAddr {
	result := make([]byte, 6)
	copy(result, dpid[2:])
	return result
}

func sendLostBuffers(dpid net.HardwareAddr, ipaddr net.IP, macaddr net.HardwareAddr, port uint16) {
	if _, found := lostBuffers[pair{dpid.String(), ipaddr.String()}]; !found {
		return
	}

	buffers := lostBuffers[pair{dpid.String(), ipaddr.String()}]
	for _, buffer := range buffers {
		msg := ofp10.NewPacketOut()
		msg.InPort = buffer.inport
		msg.BufferId = buffer.bufferId
		msg.Data = nil
		msg.AddAction(ofp10.NewActionDLDst(macaddr))
		msg.AddAction(ofp10.NewActionOutput(port))
		if sw, ok := ogo.Switch(dpid); ok {
			sw.Send(msg)
		}
	}

	delete(lostBuffers, pair{dpid.String(), ipaddr.String()})
}

func (this *L3Forwarder) PacketIn(dpid net.HardwareAddr, pkt *ofp10.PacketIn) {
	ethFrame := pkt.Data
	ip := &ipv4.IPv4{}

	// Ignore link discovery packet types.
	if ethFrame.Ethertype == 0xa0f1 || ethFrame.Ethertype == 0x88cc {
		return
	}

	if _, found := this.arpTable.Dpid(dpid); !found {
		for _, fake := range this.fakeways {
			this.arpTable.Add(dpid, net.ParseIP(fake), host{dpidToMac(dpid), ofp10.P_NONE})
		}
	}

	if ethFrame.Ethertype == eth.IPv4_MSG {
		ip = ethFrame.Data.(*ipv4.IPv4)

		this.arpTable.Add(dpid, ip.NWSrc, host{ethFrame.HWSrc, pkt.InPort})

		log.Println(dpid, pkt.InPort, "IP", ip.NWSrc, "->", ip.NWDst)

		sendLostBuffers(dpid, ip.NWSrc, ethFrame.HWSrc, pkt.InPort)

		dstaddr := ip.NWDst

		if host, found := this.arpTable.Host(dpid, dstaddr); found {
			if host.port == pkt.InPort {
				log.Println(dpid, pkt.InPort, "not sending packet for", dstaddr.String(), "back out of the input port")
			} else {
				log.Println(dpid, pkt.InPort, "installing flow for", ip.NWSrc, "=>", ip.NWDst, "out port", host.port)
			}

			msg := ofp10.NewFlowMod()
			msg.Match.InPort = pkt.InPort

			msg.Match.DLDst = ethFrame.HWDst
			msg.Match.DLSrc = ethFrame.HWSrc

			msg.Match.Wildcards = ofp10.FW_ALL ^ ofp10.FW_DL_SRC ^ ofp10.FW_DL_DST
			msg.Command = ofp10.FC_ADD

			msg.AddAction(ofp10.NewActionDLDst(host.mac))
			msg.AddAction(ofp10.NewActionOutput(host.port))

			msg.IdleTimeout = 30
			msg.HardTimeout = 20
			msg.BufferId = pkt.BufferId

			if sw, ok := ogo.Switch(dpid); ok {
				sw.Send(msg)
			}

		} else {
			if _, found := lostBuffers[pair{dpid.String(), ip.NWDst.String()}]; !found {
				lostBuffers[pair{dpid.String(), ip.NWDst.String()}] = make([]buffer, 0)
			}

			lostBuffers[pair{dpid.String(), ip.NWDst.String()}] = append(lostBuffers[pair{dpid.String(), ip.NWDst.String()}], buffer{pkt.BufferId, pkt.InPort})

			arpReq, err := arp.New(arp.Type_Request)
			if err != nil {
				panic(err)
			}
			arpReq.HWDst, _ = net.ParseMAC("ff:ff:ff:ff:ff:ff")
			arpReq.IPDst = ip.NWDst
			arpReq.HWSrc = ethFrame.HWSrc
			arpReq.IPSrc = ip.NWSrc

			e := eth.New()
			e.Ethertype = eth.ARP_MSG
			e.HWSrc = ethFrame.HWSrc
			e.HWDst, _ = net.ParseMAC("ff:ff:ff:ff:ff:ff")
			e.Data = arpReq

			log.Println(dpid, pkt.InPort, "ARPing for", arpReq.IPDst, "on behalf of", arpReq.IPSrc)

			msg := ofp10.NewPacketOut()
			msg.InPort = pkt.InPort
			msg.Data = e
			msg.AddAction(ofp10.NewActionOutput(ofp10.P_FLOOD))

			if sw, ok := ogo.Switch(dpid); ok {
				sw.Send(msg)
			}
		}
	} else if ethFrame.Ethertype == eth.ARP_MSG {
		a := ethFrame.Data.(*arp.ARP)

		log.Println(dpid, pkt.InPort, "ARP", a.Operation, a.IPSrc, "->", a.IPDst)

		if _, found := this.arpTable.Host(dpid, a.IPSrc); found {
			log.Println("RE-learned:", dpid, pkt.InPort, a.IPSrc)
		} else {
			log.Println("learned:", dpid, pkt.InPort, a.IPSrc)
		}

		this.arpTable.Add(dpid, a.IPSrc, host{ethFrame.HWSrc, pkt.InPort})

		sendLostBuffers(dpid, a.IPSrc, ethFrame.HWSrc, pkt.InPort)

		if a.Operation == arp.Type_Request {
			if host, found := this.arpTable.Host(dpid, a.IPDst); found {
				arpReply, err := arp.New(arp.Type_Reply)
				if err != nil {
					panic(err)
				}

				arpReply.HWType = a.HWType
				arpReply.ProtoType = a.ProtoType
				arpReply.HWLength = a.HWLength
				arpReply.ProtoLength = a.ProtoLength
				arpReply.HWDst = a.HWSrc
				arpReply.IPDst = a.IPSrc
				arpReply.IPSrc = a.IPDst
				arpReply.HWSrc = host.mac

				e := eth.New()
				e.Ethertype = ethFrame.Ethertype
				e.HWSrc = dpidToMac(dpid)
				e.HWDst = a.HWSrc
				e.Data = arpReply

				log.Println(dpid, pkt.InPort, "answering ARP for", arpReply.IPSrc)

				msg := ofp10.NewPacketOut()
				msg.InPort = pkt.InPort
				msg.Data = e
				msg.AddAction(ofp10.NewActionOutput(ofp10.P_IN_PORT))

				if sw, ok := ogo.Switch(dpid); ok {
					sw.Send(msg)
				}

				return
			}
		}

		log.Println(dpid, pkt.InPort, "Flooding ARP", a.IPSrc, "->", a.IPDst)

		msg := ofp10.NewPacketOut()
		msg.InPort = pkt.InPort
		msg.Data = &ethFrame
		msg.AddAction(ofp10.NewActionOutput(ofp10.P_FLOOD))

		if sw, ok := ogo.Switch(dpid); ok {
			sw.Send(msg)
		}
	}
}
func (this *L3Forwarder) ConnectionUp(dpid net.HardwareAddr) {
	log.Println("Connection up for", dpid)
}
