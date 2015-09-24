package mn

type Node interface {
	NodeName() string
	NetNs() *NetNs
	LinksCount() int
	GetCidr(peer Peer) string
	GetHwAddr(peer Peer) string
	GetState(peer Peer) string
	Release() error
	AddLink(Link) error
	GetLinks() Links
}
