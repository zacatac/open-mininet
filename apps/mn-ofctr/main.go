package main

import (
	"flag"
	"log"
	"runtime"
	"strings"

	"github.com/3d0c/ogo"
	"github.com/NodePrime/open-mininet/apps/netapps"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	f := flag.String("fakeways", "192.168.55.1,192.168.66.1", "Default gateways expected by hosts")
	name := flag.String("name", "", "netapp name, supported: l2-forwarder, l3-forwarder")
	listen := flag.String("listen", ":6633", "controller's ip:port to listen")
	apiOn := flag.String("apiOn", "", "bind addr:port to serve API, e.g. :8080")
	flag.Parse()

	fakeways := strings.Split(*f, ",")

	ctrl := ogo.NewController()

	switch *name {
	case "l2-forwarder":
		ctrl.RegisterApplication(netapps.NewL2Forwarder)
	case "l3-forwarder":
		ctrl.RegisterApplication(netapps.NewL3Forwarder(fakeways))
	case "demo":
		ctrl.RegisterApplication(netapps.NewDemoInstance)
	default:
		log.Println("No netapps selected, controller will be run in core mode")
	}

	if *apiOn != "" {
		go netapps.NewWebService(*apiOn)
	}

	ctrl.Listen(*listen)
}
