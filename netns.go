package mn

/*
#include <unistd.h>
#include <syscall.h>

int setns(int fd, int nstype) {
	return syscall(__NR_setns, fd, nstype);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"
)

const NETNS_RUN_DIR = "/var/run/netns"

type NetNs struct {
	name string
}

func NewNetNs(name string) (*NetNs, error) {
	this := &NetNs{
		name: name,
	}

	if this.Exists() {
		return this, nil
	}

	if err := this.Create(); err != nil {
		return this, err
	}

	return this, nil
}

func (this NetNs) Create() error {
	if out, err := RunCommand("ip", "netns", "add", this.name); err != nil {
		return errors.New(fmt.Sprintf("Error: %v, output: %s", err, out))
	}

	return nil
}

func (this NetNs) Exists() bool {
	out, err := RunCommand("ip", "netns", "list")
	if err != nil {
		log.Println("Error: %v, output: %s", err, out)
		return true
	}

	if strings.Contains(out, this.name) {
		return true
	}

	return false
}

func (this NetNs) Release() error {
	if this.name == "" {
		return nil
	}

	name := NETNS_RUN_DIR + "/" + this.name

	syscall.Unmount(name, syscall.MNT_DETACH)

	if err := os.Remove(name); err != nil {
		return err
	}

	return nil
}

func (this NetNs) Name() string {
	return this.name
}
