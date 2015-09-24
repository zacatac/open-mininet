package netapps

import (
	"log"
	"net"

	"github.com/3d0c/ogo/protocol/ofp10"
)

func NewDemoInstance() interface{} {
	return new(DemoInstance)
}

type DemoInstance struct{}

func (b *DemoInstance) ConnectionUp(dpid net.HardwareAddr) {
	log.Println("Switch connected:", dpid)
}

func (b *DemoInstance) ConnectionDown(dpid net.HardwareAddr) {
	log.Println("Switch disconnected:", dpid)
}

func (b *DemoInstance) PacketIn(dpid net.HardwareAddr, pkt *ofp10.PacketIn) {
	log.Println("PacketIn message received from:", dpid, "len:", pkt.Len(), "datalen:", pkt.Data.Len(), "hwsrc:", pkt.Data.HWSrc, "hwdst:", pkt.Data.HWDst, pkt.Data.Ethertype)
}
