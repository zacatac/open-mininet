package mn

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	random "github.com/Pallinder/go-randomdata"
)

func firstrealhw() net.Interface {
	empty := net.Interface{HardwareAddr: []byte{0, 0, 0, 0, 0, 0}}
	interfaces, err := net.Interfaces()
	if err != nil {
		return empty
	}

	for i, _ := range interfaces {
		if interfaces[i].HardwareAddr.String() != "" {
			return interfaces[i]
		}
	}

	return empty
}

func RunCommand(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).CombinedOutput()
	if err != nil {
		return string(out), err
	}

	return string(out), nil
}

// tmp stuff, refactor it

func hostname(n ...int) string {
	num := 200
	if len(n) > 0 {
		num = n[0]
	}

	return fmt.Sprintf("host-%d", random.Number(num))
}

func HostName(n ...int) string {
	return hostname(n...)
}

func switchname(n ...int) string {
	num := 200
	if len(n) > 0 {
		num = n[0]
	}

	return fmt.Sprintf("switch-%d", random.Number(num))
}

func SwitchName(n ...int) string {
	return switchname(n...)
}

func FullPathFor(cmd string) string {
	pathlist := os.Getenv("PATH")

	for _, path := range strings.Split(pathlist, ":") {
		_, err := os.Stat(path + "/" + cmd)
		if os.IsNotExist(err) {
			continue
		}

		return path + "/" + cmd
	}

	return ""
}
