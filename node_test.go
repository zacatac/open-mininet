package mn

import (
	"log"
	"os"
	"strings"
	"testing"
)

func init() {
	_, err := os.Stat(os.Getenv("GOPATH") + "/bin/mn-vmmock")
	if os.IsNotExist(err) {
		log.Println("Test cases depend from some binaries. Please do \"go install ./...\" before")
		os.Exit(1)
	}

	path := FullPathFor("ip")
	if path == "" {
		log.Println("ip command not found in the PATH. Please check that iproute2 utility has been installed")
		os.Exit(1)
	}

	out, _ := RunCommand(FullPathFor("ip"), "netns", "help")

	if !strings.Contains(out, "identify") {
		log.Println("iproute2 utility doesn't support netns identify command, please update it.")
		os.Exit(1)
	}
}

func TestNewHost(t *testing.T) {
	h, err := NewHost(hostname(65535))
	if err != nil {
		t.Fatal(err)
	}

	defer h.Release()

	// check, namespace has been created
	_, err = os.Stat(NETNS_RUN_DIR + "/" + h.NetNs().Name())
	if os.IsNotExist(err) {
		t.Fatal("Expected netns not found in", NETNS_RUN_DIR+h.NetNs().Name())
	}
}

func TestHostRunProcess(t *testing.T) {
	h, err := NewHost()
	if err != nil {
		t.Fatal(err)
	}

	defer h.Release()

	p, err := h.RunProcess(os.Getenv("GOPATH")+"/bin/mn-vmmock", "-name", h.NodeName())
	if err != nil {
		t.Fatal(err)
	}

	out, err := RunCommand("ps", "-ax")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(out, "mn-vmmock") {
		t.Fatal("Expected process not found")
	}

	p.Stop()

	out, err = RunCommand("ps", "-ax")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(out, "mn-vmmock") {
		t.Fatal("Unexpected process found")
	}
}

func TestRouter(t *testing.T) {
	r, err := NewRouter()
	if err != nil {
		t.Fatal(err)
	}

	defer r.Release()

	if !isForwardingEnabled(r.NetNs().Name()) {
		t.Fatal("Expected net.ipv4.ip_forward=1 for", r.NetNs().Name())
	}
}

func TestSwitchExists(t *testing.T) {
	s1, err := NewSwitch()
	if err != nil {
		t.Fatal(err)
	}

	defer s1.Release()

	if !s1.Exists() {
		t.Fatal("Expecting switch", s1.NodeName(), "exists")
	}

	s2 := Switch{Name: "xcvxcvcxv"}
	if s2.Exists() {
		t.Fatal("Expecting switch", s2.NodeName(), "does not exist")
	}
}

func TestNetNsExists(t *testing.T) {
	ns1, err := NewNetNs(hostname())
	if err != nil {
		t.Fatal(err)
	}

	defer ns1.Release()

	if !ns1.Exists() {
		t.Fatal("Expected netns", ns1.Name(), "exists")
	}

	ns2 := NetNs{name: "xcvxcvxc444"}

	if ns2.Exists() {
		t.Fatal("Expected netns", ns2.Name(), "does not exist")
	}
}

func isForwardingEnabled(netns string) bool {
	out, err := RunCommand(FullPathFor("ip"), "netns", "exec", netns, "sysctl", "net.ipv4.ip_forward")
	if err != nil {
		log.Println(err)
		return false
	}

	return strings.Contains(out, "1")
}
