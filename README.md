## open-mininet
Open source Go implementation of mininet and openflow controller examples.

#### Features
- Creating network topology from **hosts** and **switches**
- **Host** with more than one network interface can be a linux router
- **Switches** can be interconnected with each other
- **Hosts** are isolated in network namespaces
- **Hosts**  can run processess
- Processess can be limited by cgroups
- JSON defined scheme


### Prerequisites

Install libcgroups and libcgroups-dev.  
Ubuntu commands:

```
apt-get install libcgroup-dev libcgroup1
```

#### Tests

Because tests depend from some utilities in apps directory, install package before test

```sh
go install ./...
```

Then, all tests should pass

```
go test
```


## JSON defined network scheme

Configuration could be __imported__ and __exported__ as a JSON. Take a look at [example.json](apps/example.json) to be familiar with its structure. 
All we need now to restore this configuration is to run following API

```go
	scheme, err := NewSchemeFromJson("apps/example.json")
	if err != nil {
		t.Fatal(err)
	}
	
	scheme.Recover()
```
And switches, hosts, namespaces, links, cgroups and processess will be created, if they don't exist.  

### Processess and Cgroups

If a **Host** record has a "Cgroup" field, like __net1-h1__ host from [example.json](apps/example.json):

```json

"Hosts": [
            {
                  "Name": "net1-h1",
                  "Cgroup": {
                        "Name": "net1-h1",
                        "Controllers": [
							...
                        ]
                  },
            ...
```

Each process from "Procs" list will be started as follows:

```sh
cgexec -g ctrlname,ctrlname:net1-h1 ip netns net1-h1 exec command args...
```

If there is no cgroups, execution will be as simple as:

```sh
ip netns net1-h1 exec command args...
```

### Links and interconnection
**Switches** ports have two type:  

-  general for host connection
-  patch port for switch connection

General port is just a left side of __veth__ pair, created like:

```sh
ip link add name net1-h1-eth0 type veth peer name eth0 netns net1-h1
```

right part __eth0__ moved to host "h1" namespace. So we become a link

```sh
[root] net1-h1-eth0 <------> [h1] eth0
```

Let's see this pair in our exmaple.json:

```json
      "Switches": [
            {
                  "Name": "s1",
                  "Ports": [
                        {
                              "Cidr": "noip",
                              "HwAddr": "08:00:27:95:1e:bf",
                              "Name": "net1-h1-eth0",
                              "NodeName": "s1",
                              "NetNs": "",
                              "State": "UP",
                              "Routes": null,
                              "PeerName": "",
                              "Peer": {
                                    "Name": "net1-h1-eth0",
                                    "IfName": "eth0",
                                    "NodeName": "net1-h1"
                              }
                        },
			...
		
      "Hosts": [
                  "Links": [
                        {
                              "Cidr": "192.168.55.2/24",
                              "HwAddr": "08:00:27:a2:eb:aa",
                              "Name": "eth0",
                              "NodeName": "net1-h1",
                              "NetNs": "net1-h1",
                              "State": "UP",
                              "Routes": [
                                    {
                                          "Dst": "0.0.0.0/0",
                                          "Gw": "192.168.55.1"
                                    }
                              ],
                              "PeerName": "",
                              "Peer": {
                                    "Name": "s1-net1-h1-eth0",
                                    "IfName": "net1-h1-eth0",
                                    "NodeName": "s1"
                              }
                        }
                  ]
      

```
Pretty straightforward.

**Patch port** is very similar to normal link except it's created as follows (according example.json):

```sh
ovs-vsctl add-port s1 s1-patch-port4
ovs-vsctl set interface s1-patch-port4 type=patch
ovs-vsctl set interface s1-patch-port4 "options:peer=s2-patch-port0"
```


## Examples
- __Simple Topo__  
  creates 10 namespaced hosts connected to the vSwitch and pings each other from each other

```
go test -run=TestTopoSimple
```
- __Machines__  
  creates 10 namespaced "virtual machines", which is a simple go program, which expose IPMI emulator. And are stopped with `IPMIPowerOff` call.
  
```
go test -run=TestMachines
```

## mn-ctl 
Simple control utility. It has a command line interface with history and some autocompletion.  
Sample walkthrough:  

```sh
> new switch
Switch switch-142 created
> new host
Host host-83 created
> new link switch-142 host-83
[Link] switch-142 host-83-eth0 192.168.55.1/24 <---> host-83 eth0 192.168.55.2/24
> dump
Switch: switch-142
	host-83-eth0 [UP] <------> eth0 [192.168.55.2/24:72:bd:eb:04:45:16] [UP]
Disconnected hosts:
>
```

### Multiple networks and linux router example

__NewRouter__ call exposes a Node with forwarder capability. You can easily reproduce this example:  
Start __ctl__ utility and copy/paste following commands:

```sh
new switch s1
new host net1-h1
new host net2-h1
new router r1
new link s1 net1-h1 {"Cidr":"noip"} {"Cidr":"192.168.55.2/24","Routes":[{"Dst":"0.0.0.0/0","Gw":"192.168.55.1"}]}
new link s1 net2-h1 {"Cidr":"noip"} {"Cidr":"192.168.66.2/24","Routes":[{"Dst":"0.0.0.0/0","Gw":"192.168.66.1"}]}
new link s1 r1 {"Cidr":"noip"} {"Cidr":"192.168.55.1/24"}
new link s1 r1 {"Cidr":"noip"} {"Cidr":"192.168.66.1/24"}

```

After that, __dump__ command should return following:

```sh
> dump
Switch: s1
	net1-h1-eth0 [UP] <------> eth0 [192.168.55.2/24 08:00:27:88:60:42] [UP]
	net2-h1-eth0 [UP] <------> eth0 [192.168.66.2/24 08:00:27:89:c3:14] [UP]
	r1-eth0 [UP] <------> eth0 [192.168.55.1/24 08:00:27:8b:7d:fb] [UP]
	r1-eth1 [UP] <------> eth1 [192.168.66.1/24 08:00:27:04:3a:65] [UP]
```

So, the __r1__ node has two links into both networks and has __net.ipv4.ip_forward=1__. The links __Routes__ options exposes route command with corresponding values.  
Now, check it:

```sh
> net1-h1 route -n
Kernel IP routing table
Destination     Gateway         Genmask         Flags Metric Ref    Use Iface
0.0.0.0         192.168.55.1    0.0.0.0         UG    0      0        0 eth0
192.168.55.0    0.0.0.0         255.255.255.0   U     0      0        0 eth0
```

And, finally, ping foreign network:

```sh
> net1-h1 ping -c1 192.168.66.2
PING 192.168.66.2 (192.168.66.2) 56(84) bytes of data.
64 bytes from 192.168.66.2: icmp_seq=1 ttl=63 time=1.37 ms
```

### Switches interconnection
__NewSwitchLink__ call prepares a link pair for switch __AddPatchPort__ method, then a set of *ovs* commands for patch interface are exposed.  
__Ctl__ example:

```sh
new switch s1
new host s1-host1
new link s1 s1-host1 {"Cidr":"noip"} {"Cidr":"192.168.55.10/24"}

new switch s2
new host s2-host1
new link s2 s2-host1 {"Cidr":"noip"} {"Cidr":"192.168.55.11/24"}

new link s1 s2
```

so, you will get the following configuration:

```sh
> dump
Switch: s1
	s1-host1-eth0 [UP] <------> eth0 [192.168.55.10/24:08:00:27:3a:fa:58] [UP]
	s1-patch-port1 [] <------>  [:] []
Switch: s2
	s2-host1-eth0 [UP] <------> eth0 [192.168.55.11/24:08:00:27:d6:a9:29] [UP]
	s2-patch-port1 [] <------>  [:] []
```

check it with ping:

```sh
> s1-host1 ping -c1 192.168.55.11
PING 192.168.55.11 (192.168.55.11) 56(84) bytes of data.
64 bytes from 192.168.55.11: icmp_seq=1 ttl=64 time=1.54 ms
```

### Working with processes
There is two command for process executing. First one, you've been familiar with — just write command after hostname. Process isn't detached from console and all output goes to the stdout. E.g.:

```sh
> import example.json
> net1-h1 ping -c1 192.168.66.2
PING 192.168.66.2 (192.168.66.2) 56(84) bytes of data.
64 bytes from 192.168.66.2: icmp_seq=1 ttl=63 time=0.047 ms

--- 192.168.66.2 ping statistics ---
1 packets transmitted, 1 received, 0% packet loss, time 0ms
rtt min/avg/max/mdev = 0.047/0.047/0.047/0.000 ms
>
```
Second one is the command **start**. E.g.:

```sh
> net1-h1 start ping -c1000 192.168.66.2
Started [/usr/bin/cgexec -g cpu,memory:net1-h1 /usr/sbin/ip netns exec net1-h1 ping -c1000 192.168.66.2] All output goes to /tmp/output.1440933646
>
```

Stderr and Stdout will be redirected to temporary file. Let's see our processes list

```sh
> net1-h1 ps
    0 /bin/ping -c100 192.168.66.2
    0 /bin/ping -c200 192.168.66.2
30057 ping -c1000 192.168.66.2
>
```

Processes with zero pid were imported but not recovered, because we didn't run **recover** command, so only the last one are running.  
To see the process output type:

```sh
> net1-h1 proc output 30057
PING 192.168.66.2 (192.168.66.2) 56(84) bytes of data.
64 bytes from 192.168.66.2: icmp_seq=1 ttl=63 time=0.043 ms
64 bytes from 192.168.66.2: icmp_seq=2 ttl=63 time=0.160 ms
					...
```

To stop the process use **proc stop** command. E.g.:

```sh
> net1-h1 proc stop 30057
E0830 11:25:06.037889 30040 host.go:156] Process [30057] [/usr/bin/cgexec -g cpu,memory:net1-h1 /usr/sbin/ip netns exec net1-h1 ping -c1000 192.168.66.2] finished with true, exit status 0
```

## API Walkthrought
Interconnect two hosts with the switch, ping and release the scheme.

```go
package main

import (
	mn "github.com/NodePrime/open-mininet"
	"time"
)

func main() {
	// creating an instance of Scheme struct, which is just a
	// a storage for futher nodes
	scheme := mn.NewScheme()

	defer scheme.Release()

	// create new host h1
	host1, err := mn.NewHost("h1")
	if err != nil {
		panic(err)
	}

	// create new switch with random name
	sw, err := mn.NewSwitch()
	if err != nil {
		panic(err)
	}

	// interconnect nodes
	pair := mn.NewLink(sw, host1, mn.Link{Cidr: "noip"}, mn.Link{Cidr: "192.168.44.1/24"})

	// physically create link
	if err := pair.Create(); err != nil {
		panic(err)
	}

	// apply cidr, routes, bring interfaces up
	pair, err = pair.Up()
	if err != nil {
		panic(err)
	}

	sw.AddLink(pair.Left)
	host1.AddLink(pair.Right)

	scheme.AddNode(host1)
	scheme.AddNode(sw)

	// repeat for host2

	host2, err := mn.NewHost("h2")
	if err != nil {
		panic(err)
	}

	pair2 := mn.NewLink(sw, host2, mn.Link{Cidr: "noip"}, mn.Link{Cidr: "192.168.44.2/24"})
	if err := pair2.Create(); err != nil {
		panic(err)
	}
	pair2, err = pair2.Up()
	if err != nil {
		panic(err)
	}

	sw.AddLink(pair2.Left)
	host2.AddLink(pair2.Right)

	scheme.AddNode(host2)

	// now we can run some command

	host1.RunProcess("ping", "-c1", "192.168.44.2")

	time.Sleep(time.Second * 1)
}

```

## Openflow network applications

Do the **go get -t ./...** to install dependencies.

You can find all network application inside apps/netpapps folder:

- **apps/netapps/l2-forwarder.go**  
  Simple l2 learning.

- **apps/netapps/l3-forwarder.go**  
  l3 routing between different subnetworks inside one or multiple switches. Beware of this example, the fake routes are hardcoded and coupled with the schemes/l3.json. Just an example.
  
- If no apps specified, controller will be run in core mode.
  
And **mn-ofctr** utility to control them.

There is a corresponding json scheme for each network application, located in __apps/schemes__. Suffix `-multi` means two switches in scheme, connected with a patch link. To get it work, do the following:

- stop default controller, if it exists  
  `service openvswitch-controller stop` or `killall -9 ovs-controller`
  
- run cleanup script, `apps/cleanup.sh` (optionally)
- run **mn-ctl**, import and recover the scheme, e.g.:

```
~ go run apps/mn-ctl/main.go
> import apps/scheme/l3.json
> recover
> ctrl+c
```
- start corresponding network application

```
go run apps/mn-ofctr/main.go -name=l3-forwarder
```
Use `-v=4 -logtostderr=true` for verbose output.

- check it works. From another terminal do ping.

```
~ ip netns exec net1-h1 ping 192.168.66.2
```

Please checkout network schemes, there is a field "Controller" in the switch object, this is a controller's address, which is tcp:0.0.0.0:6633 by default, so you can test altogether inside one host.

### Openflow web service

To start web service run, specify `apiOn=` option of the **mn-ofctr** utility, e.g:

```
~ go run apps/mn-ofctr/main.go -apiOn=":8080"
```

Methods:

- **GET /switches**  
  Returns all switches connected to the controller

- **POST /switches/:dpid/flows**  
  Data prototype:  

```javascript
	{
		"FlowMods": [
			{
				"Match": { 
					// Match options
				}, 
				
				"Actions": [
					// Actions list
				]
			}
		]
	}
```

E.g.:

```
curl -i -XPOST -d '{"FlowMods":[{"Match": { "DLSrc":"00:11:22:33:44:55", "DLVLAN":5, "TPSrc":10, "DLType":555}, "Actions":[{"Type":"OFPAT_OUTPUT", "Value":"P_FLOOD"}]}]}'  http://localhost:8080/switches/00:00:d6:41:26:c9:e9:45/flows
```
