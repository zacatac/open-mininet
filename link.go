package mn

import (
	"errors"
	"fmt"
	"net"
	"reflect"

	"github.com/NodePrime/open-mininet/pool"
)

type Pair struct {
	Left  Link
	Right Link
}

type Route struct {
	Dst string
	Gw  string
}

type Peer struct {
	Name     string
	IfName   string
	NodeName string
}

type Link struct {
	Cidr      string
	HwAddr    string
	Name      string
	NodeName  string
	NetNs     string
	State     string
	Routes    []Route
	PeerName  string
	Peer      Peer
	patch     bool
	ForceRoot bool `json:"-"`
}

const noip = "noip"

// NewLink(n1, n2, [link1 properties, link2 properties])
// property which isn't specified will be generated
func NewLink(left, right Node, refs ...Link) Pair {
	result := Pair{}

	switch len(refs) {
	case 1:
		result = Pair{Left: refs[0], Right: Link{}}
	case 2:
		result = Pair{Left: refs[0], Right: refs[1]}
	default:
		break
	}

	if reflect.TypeOf(left).Elem() == reflect.TypeOf(right).Elem() && reflect.TypeOf(left).Elem() == reflect.TypeOf(Switch{}) {
		result.Left = result.Left.SetNodeName(left).SetName(left, "pp").SetPatch()
		result.Right = result.Right.SetNodeName(right).SetName(right, "pp").SetPatch()
	} else {
		result.Left = result.Left.SetCidr().SetHwAddr().
			SetNetNs(left).SetName(right, "eth").SetNodeName(left).SetState("DOWN").SetRoute()

		result.Right = result.Right.SetCidr().SetHwAddr().
			SetNetNs(right).SetName(right, "eth").SetNodeName(right).SetState("DOWN").SetRoute()
	}

	result.Left = result.Left.SetPeer(result.Right)
	result.Right = result.Right.SetPeer(result.Left)

	return result
}

// type link Link
//
// func (this *Link) UnmarshalJSON(b []byte) error {
// 	l := &link{}

// 	if err := json.Unmarshal(b, l); err != nil {
// 		return err
// 	}

// 	*this = *(*Link)(l)
// 	return nil
// }

func (this Pair) ByNodeName(n Node) Link {
	if this.Left.NodeName == n.NodeName() {
		return this.Left
	} else {
		return this.Right
	}
}

func (this Pair) Create() error {
	command := []string{"link", "add", "name", this.Left.Name, "type", "veth", "peer", "name", this.Right.Name}

	if this.Right.NetNs != "" {
		command = append(command, "netns", this.Right.NetNs)
	}

	if out, err := RunCommand("ip", command...); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	if this.Left.NetNs != "" {
		if err := this.Left.MoveToNs(this.Left.NetNs); err != nil {
			return err
		}
	}

	return nil
}

func (this Pair) Up() (Pair, error) {
	if this.Left.patch {
		return this, nil
	}

	if err := this.Left.ApplyCidr(); err != nil {
		return this, errors.New(fmt.Sprint("Unable to Left.ApplyCidr, error:", err))
	}

	if err := this.Right.ApplyCidr(); err != nil {
		return this, errors.New(fmt.Sprint("Unable to Right.ApplyCidr, error:", err))
	}

	if err := this.Left.Up(); err != nil {
		return this, errors.New(fmt.Sprint("Unable to Left.Up(), error:", err))
	}

	if err := this.Right.Up(); err != nil {
		return this, errors.New(fmt.Sprint("Unable to Right.Up(), error:", err))
	}

	if err := this.Right.ApplyRoutes(); err != nil {
		return this, errors.New(fmt.Sprint("Unable to ApplyRoutes(), error:", err))
	}

	this.Left = this.Left.SetState("UP")
	this.Right = this.Right.SetState("UP")

	fmt.Println("[Link]", this.Left.NodeName, this.Left.Name, this.Left.Cidr, "<--->", this.Right.NodeName, this.Right.Name, this.Right.Cidr)

	return this, nil
}

func (this Pair) Release() {
	this.Left.Release()
	this.Right.Release()
}

func (this Pair) IsPatch() bool {
	return this.Left.patch
}

func (this Link) Release() {
	command := []string{"ip", "link", "delete", this.Name}

	if this.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", this.NetNs}, command...)
	}

	RunCommand(command[0], command[1:]...)
}

func (this Link) ApplyMac() error {
	if out, err := RunCommand("ip", "link", "set", "dev", this.Name, "address", this.HwAddr); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	return nil
}

func (this Link) Up() error {
	command := []string{"ip", "link", "set", this.Name, "up"}

	if this.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", this.NetNs}, command...)
	}

	if out, err := RunCommand(command[0], command[1:]...); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	this.State = "UP"

	return nil
}

func (this Link) ApplyCidr() error {
	if _, _, err := net.ParseCIDR(this.Cidr); err != nil {
		// omit setting ip, by passing some garbage to input
		return nil
	}

	command := []string{"ip", "addr", "add", this.Cidr, "dev", this.Name}

	if this.NetNs != "" {
		command = append([]string{"ip", "netns", "exec", this.NetNs}, command...)
	}

	if out, err := RunCommand(command[0], command[1:]...); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	return nil
}

func (this Link) ApplyRoutes() error {
	for _, route := range this.Routes {
		commands := []string{"route", "add", "-net", route.Dst, "gw", route.Gw}
		if this.NetNs != "" {
			commands = append([]string{"ip", "netns", "exec", this.NetNs}, commands...)
		}

		out, err := RunCommand(commands[0], commands[1:]...)
		if err != nil {
			return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
		}
	}

	return nil
}

func (this Link) Exists() bool {
	_, err := RunCommand("ip", "link", "show", this.Name)
	return err == nil
}

func (this Link) MoveToNs(netns string) error {
	if out, err := RunCommand("ip", "link", "set", this.Name, "netns", netns); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	return nil
}

func (this Link) SetCidr() Link {
	if this.Cidr == "" {
		this.Cidr = pool.ThePool().NextCidr()
	}

	return this
}

func (this Link) SetHwAddr() Link {
	if this.HwAddr == "" {
		this.HwAddr = pool.ThePool().NextMac(firstrealhw().HardwareAddr.String())
	}

	return this
}

// interface pairs naming rules:
//   left (host node)          {peer_host}-{prefix}X
//   right (namespaced node)   ethX
func (this Link) SetName(n Node, prefix string) Link {
	if this.Name != "" {
		return this
	}

	if this.NetNs != "" {
		this.Name = fmt.Sprintf("veth%d", n.LinksCount())
	} else {
		this.Name = fmt.Sprintf("%s-%s%d", n.NodeName(), prefix, n.LinksCount())
	}

	return this
}

func (this Link) SetNodeName(n Node) Link {
	if this.NodeName == "" {
		this.NodeName = n.NodeName()
	}

	return this
}

func (this Link) SetNetNs(n Node) Link {
	if this.NetNs == "" && n.NetNs() != nil {
		this.NetNs = n.NetNs().Name()
	}

	return this
}

func (this Link) SetState(s string) Link {
	this.State = s
	return this
}

func (this Link) SetRoute() Link {
	return this
}

func (this Link) SetPeer(l Link) Link {
	this.Peer.IfName = l.Name
	this.Peer.NodeName = l.NodeName
	this.Peer.Name = l.Name

	return this
}

func (this Link) SetPatch() Link {
	if !this.ForceRoot {
		this.patch = true
	}

	return this
}

func (this Link) Ip() string {
	ip, _, _ := net.ParseCIDR(this.Cidr)
	return ip.String()
}

type Links []Link

func (this Links) LinkByPeer(peer Peer) Link {
	for _, link := range this {
		if link.NodeName == peer.NodeName && link.Name == peer.IfName {
			return link
		}
	}

	return Link{}
}
