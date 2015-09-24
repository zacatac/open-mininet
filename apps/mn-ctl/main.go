package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	mn "github.com/NodePrime/open-mininet"
	"github.com/NodePrime/open-mininet/pool"
	"github.com/peterh/liner"
)

var (
	history_fn = "/tmp/.liner_history"
	names      = []string{"help", "new", "new host", "new switch", "new link", "new router", "dump-json", "import", "recover", "release", "show hosts", "show switches"}
)

var generalHelpTest = `
Generat help topic.
Host always has its own namespace, switch hasn't. Host's netns is equal to it's name.
Commands:
  new host   [name]     Creates new host instance
  new switch [name]     Creates new switch instance
  
  new link   [nodeLeft, nodeRigh, LeftLinkOptions, RightLinkOptions]
             Right node should be a namespaced host.
             Options:
                Cidr:   valid_cidr or noip literal
                Name:   interface name
                HwAddr: interface address

                E.g.:
                    new link switch1, host1
                Control interface for switch:
                    new link switch1 host1 {"Cidr":"noip", "Name":"ctrl0"} {"Cidr":"192.168.55.200/24", "Name":"ctrl1"}

  new router [name]     Create router, same as host, but with forwarding enabled
  dump                  Dump as a plain text
  dump-json             Dump as a json
  show hosts            Print hosts
  show switches         Print switches
  import {file.json}    Import json scheme 
  
  Host command:
  hostname ps           Show processess associated with host
  hostname proc output  {pid} Show process output
  hostname proc stop    {pid} Stop process
`

func help(commands ...string) {
	if len(commands) == 1 {
		fmt.Println(generalHelpTest)
	}
}

// var hosts map[string]*mn.Host = make(map[string]*mn.Host)
// var switches map[string]*mn.Switch = make(map[string]*mn.Switch)

var scheme *mn.Scheme = mn.NewScheme()

func newNode(commands ...string) {
	switch commands[0] {
	case "host":
		var name string
		if len(commands) == 2 {
			name = commands[1]
		}

		h, err := mn.NewHost(name)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println("Host", h.NodeName(), "created")
		scheme.AddNode(h)

	case "router":
		var name string
		if len(commands) == 2 {
			name = commands[1]
		}

		h, err := mn.NewRouter(name)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println("Host", h.NodeName(), "created")
		scheme.AddNode(h)

	case "switch":
		var name string
		if len(commands) == 2 {
			name = commands[1]
		}

		s, err := mn.NewSwitch(name)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println("Switch", s.NodeName(), "created")

		scheme.AddNode(s)
		names = append(names, s.NodeName())

	case "link":
		var left, right mn.Link

		args := commands[1:]
		if len(args) < 2 {
			log.Println("At least two nodes required, e.g.: new link a,b")
			return
		}

		n1 := args[0]
		n2 := args[1]

		if len(args) >= 3 && args[2] != "" {
			if err := json.Unmarshal([]byte(args[2]), &left); err != nil {
				log.Println(err)
				return
			}
		}

		if len(args) >= 4 && args[3] != "" {
			if err := json.Unmarshal([]byte(args[3]), &right); err != nil {
				log.Println(err)
				return
			}
		}

		node1, found := scheme.GetNode(n1)
		if !found {
			log.Println("No such node:", n1, "create it first")
			return
		}

		node2, found := scheme.GetNode(n2)
		if !found {
			log.Println("No such node:", n1, "create it first")
			return
		}

		pair := mn.NewLink(node1, node2, left, right)

		if err := pair.Create(); err != nil {
			log.Println("Unable to create pair:", err)
			return
		}

		pair, err := pair.Up()
		if err != nil {
			log.Println("Can't bring it up,", err)
		}

		node1.AddLink(pair.Left)
		node2.AddLink(pair.Right)

		if pair.IsPatch() {
			fmt.Println("[Patch]", node1.NodeName(), "<--->", node2.NodeName())
		}

		return
	}
}

func dump() {
	for _, s := range scheme.Switches {
		fmt.Println("Switch:", s.NodeName())
		for _, port := range s.Ports {
			peer, found := scheme.GetNode(port.Peer.NodeName)
			if !found {
				fmt.Printf("\t%s [%s] <------ [no peer!]\n", port.Name, port.State)
				continue
			}

			fmt.Printf("\t%s [%s] <------> %s [%s, %s] [%s]\n", port.Name, port.State, peer.NodeName(), peer.GetCidr(port.Peer), peer.GetHwAddr(port.Peer), peer.GetState(port.Peer))
		}
	}

	fmt.Println("Disconnected hosts:")
	for _, h := range scheme.Hosts {
		if h.LinksCount() == 0 {
			fmt.Printf("\t%s\n", h.NodeName())
		}
	}

}

func hostCommand(commands []string) {
	host, found := scheme.GetHost(commands[0])
	if !found {
		log.Println("Host", host, "not found in scheme")
		return
	}

	switch commands[1] {
	case "ps":
		for _, process := range host.Procs {
			fmt.Printf("%5d %s %s\n", process.GetPid(), process.Command, strings.Join(process.Args, " "))
		}

	case "start":
		_, err := host.RunProcess(commands[2:]...)
		if err != nil {
			log.Println("Error running process:", err)
		}

	case "proc":
		if len(commands) < 4 {
			log.Println("Please provide a pid of process to show")
			break
		}

		pid, err := strconv.Atoi(commands[3])
		if err != nil {
			log.Println("Wrong pid", commands[3])
			break
		}

		proc := host.Procs.GetByPid(pid)
		if proc == nil {
			log.Println("Can't find process", commands[3])
			break
		}

		if commands[2] == "stop" {
			if err := proc.Stop(); err != nil {
				log.Println(err)
			}

			break
		}

		if commands[2] == "output" {
			out, err := ioutil.ReadFile(proc.Output)
			if err != nil {
				log.Println("Can't open process output file", proc.Output)
				break
			}

			fmt.Println(string(out))
		}

	default:
		commands = append([]string{"netns", "exec"}, commands...)
		out, err := mn.RunCommand("ip", commands...)
		if err != nil {
			log.Println("Error:", err, "Output:", out)
		}

		fmt.Println(out)
	}

}

func init() {
	pool.ThePool("192.168.55.1/24")
}

func main() {
	line := liner.NewLiner()
	defer line.Close()

	line.SetCtrlCAborts(true)

	line.SetCompleter(func(line string) (c []string) {
		for _, n := range names {
			if strings.HasPrefix(n, strings.ToLower(line)) {
				c = append(c, n)
			}
		}
		return
	})

	if f, err := os.OpenFile(history_fn, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		line.ReadHistory(f)
		f.Close()
	}

	defer func() {
		if f, err := os.Create(history_fn); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	for {
		input, err := line.Prompt("> ")
		if err != nil {
			if err == liner.ErrPromptAborted {
				log.Println("Aborted")
				break
			} else {
				log.Println("Error reading line: ", err)
				break
			}
		}

		line.AppendHistory(input)

		commands := strings.Split(input, " ")

		if _, found := scheme.GetHost(commands[0]); found {
			hostCommand(commands)
		}

		switch commands[0] {
		case "help":
			if len(commands) > 1 {
				help(commands[1:]...)
			} else {
				help(commands[0])
			}

		case "new":
			if len(commands) > 1 {
				newNode(commands[1:]...)
			} else {
				log.Println("Bad arguments")
			}

		case "dump":
			dump()

		case "dump-json":
			fmt.Println(scheme)

		case "import":
			if len(commands) > 1 {
				tmp, err := mn.NewSchemeFromJson(commands[1])
				if err != nil {
					log.Println(err)
					break
				}

				scheme = tmp

				fmt.Println("Scheme", commands[1], "imported. User 'recover' command to apply it.")
			} else {
				log.Println("Bad arguments")
			}

		case "recover":
			if scheme != nil {
				scheme.Recover()
			}

		case "release":
			if scheme != nil {
				scheme.Release()
			}

		case "show":
			if len(commands) == 1 {
				log.Println("Bad arguments")
				break
			}

			if commands[1] == "hosts" {
				for _, node := range scheme.Hosts {
					fmt.Println(node.NodeName())
				}
				break
			}

			if commands[1] == "switches" {
				for _, node := range scheme.Switches {
					fmt.Println(node.NodeName)
				}
			}
		}

	}

}
