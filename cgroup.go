package mn

import (
	"encoding/json"
	"log"
	"strings"

	"github.com/NodePrime/open-mininet/cgroup"
)

type Cgroup struct {
	*cgroup.Cgroup
	Name        string
	Controllers []Controller
}

type Controller struct {
	Name   string
	Params []Set
}

type Set struct {
	Key   string
	Value interface{}
}

// Working with cgroups
// 1. Init
// 2. Init Cgroup struct
// 3. Set controllers
// 4. Physically create cgroup
// 5. Set controllers values
func NewCgroup(name string) (*Cgroup, error) {
	this := &Cgroup{Name: name}

	cgroup.Init()

	this.Cgroup = cgroup.NewCgroup(name)

	return this, nil
}

func (this *Cgroup) UnmarshalJSON(b []byte) error {
	type tmp Cgroup
	cg := tmp{}

	cgroup.Init()

	if err := json.Unmarshal(b, &cg); err != nil {
		return err
	}

	this.Name = cg.Name
	this.Cgroup = cgroup.NewCgroup(cg.Name)

	if err := this.SetControllers(cg.Controllers); err != nil {
		return err
	}

	if err := this.Cgroup.Create(); err != nil {
		return err
	}

	if err := this.SetParams(cg.Controllers); err != nil {
		return err
	}

	this.Controllers = cg.Controllers

	return nil
}

func (this *Cgroup) SetControllers(controllers []Controller) error {
	for _, controller := range controllers {
		_, err := this.AddController(controller.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *Cgroup) SetParams(controllers []Controller) error {
	for _, controller := range controllers {
		ctrl := this.GetController(controller.Name)

		for _, cv := range controller.Params {
			switch t := cv.Value.(type) {
			case string:
				if err := ctrl.SetValueString(cv.Key, cv.Value.(string)); err != nil {
					return err
				}
			case float64:
				if err := ctrl.SetValueInt64(cv.Key, (int64)(cv.Value.(float64))); err != nil {
					return err
				}
			case bool:
				if err := ctrl.SetValueBool(cv.Key, cv.Value.(bool)); err != nil {
					return err
				}
			default:
				log.Println("Unexpected type:", t)
			}
		}
	}

	return nil
}

func (this *Cgroup) Release() {
	if this != nil {
		this.DeleteExt(cgroup.DeleteRecursive)
	}
}

func (this *Cgroup) CgExecCommand() []string {
	var groups string

	command := []string{FullPathFor("cgexec"), "-g"}

	for _, controller := range this.Controllers {
		groups += controller.Name + ","
	}

	groups = strings.TrimRight(groups, ",") + ":" + this.Name

	return append(command, groups)
}
