package mn

import (
	"testing"
)

func TestCgroupImport(t *testing.T) {
	scheme, err := NewSchemeFromJson(exampleScheme)
	if err != nil {
		t.Fatal(err)
	}

	defer scheme.Release()

	host, found := scheme.GetHost("net1-h1")
	if !found {
		t.Fatal("Expected host net1-h1 not found")
	}

	c := host.Cgroup.GetController("cpu")

	v, err := c.GetValueInt64("cfs_period_us")
	if err != nil {
		t.Fatal(err)
	}

	if v != 100000 {
		t.Fatal("Expected value=100000, obtained:", v)
	}

	if len(host.Cgroup.Controllers) == 0 {
		t.Fatal("Expected non zero length")
	}
}

func TestCgroupCtrl(t *testing.T) {
	cg, err := NewCgroup("asdf5")
	if err != nil {
		t.Fatal(err)
	}

	cs := []Controller{
		{
			Name: "cpu",
			Params: []Set{
				{
					Key:   "cfs_period_us",
					Value: "100000",
				},
			},
		},
		{
			Name: "memory",
			Params: []Set{
				{
					Key:   "limit_in_bytes",
					Value: "2G",
				},
			},
		},
	}

	if err := cg.SetControllers(cs); err != nil {
		t.Fatal(err)
	}

	if err := cg.Create(); err != nil {
		t.Fatal(err)
	}

	defer cg.Release()

	if err := cg.SetParams(cs); err != nil {
		t.Fatal(err)
	}

	c := cg.GetController("cpu")

	v, err := c.GetValueInt64("cfs_period_us")
	if err != nil {
		t.Fatal(err)
	}

	if v != 100000 {
		t.Fatal("Expected value=100000, obtained:", v)
	}

	c = cg.GetController("memory")
	vs, err := c.GetValueString("limit_in_bytes")
	if err != nil {
		t.Fatal(err)
	}

	if vs != "2G" {
		t.Fatal("Expected value=2G, obtained:", v)
	}
}
