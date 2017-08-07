package members

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	gcfg "gopkg.in/gcfg.v1"
)

type vmwareMembers struct {
	instances []Instance
	instance  Instance
}

func (m vmwareMembers) GetInstances() []Instance {
	return m.instances
}

func (m vmwareMembers) GetLocalInstance() Instance {
	return m.instance
}

// NewVMware returns the Members this local instance belongs to.
func NewVMware(configLocation, env, role string) (Members, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	f, err := os.Open(configLocation)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	cfg, err := readConfig(f)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	c, err := NewClient(ctx, cfg)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer c.Logout(ctx)

	instances, err := findAllInstances(ctx, c, env, role)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	instance, err := findThisInstance(cfg, instances)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	members := vmwareMembers{
		instances: instances,
		instance:  *instance,
	}

	return members, nil
}

func findAllInstances(ctx context.Context, c *govmomi.Client, env, role string) ([]Instance, error) {
	m := view.NewManager(c.Client)

	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	defer v.Destroy(ctx)

	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	var vms []mo.VirtualMachine

	// Does restricting the scope for the fields we're after make it faster?
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"config.name", "config.extraConfig", "summary.runtime", "summary.guest"}, &vms)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var instances []Instance

	var matched []mo.VirtualMachine
	for _, vm := range vms {
		if matchesTag(vm, "tags_environment", env) && matchesTag(vm, "tags_role", role) {
			matched = append(matched, vm)
		}
	}

	for _, vm := range matched {
		if vm.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn {
			instances = append(instances, Instance{
				InstanceID: vm.Config.Name,
				PrivateIP:  vm.Summary.Guest.IpAddress,
			})
		}
	}

	return instances, nil
}

func matchesTag(vm mo.VirtualMachine, tag string, match string) bool {
	if vm.Config != nil {
		for _, config := range vm.Config.ExtraConfig {
			if config.GetOptionValue().Key == tag && config.GetOptionValue().Value == match {
				return true
			}
		}
	}
	return false
}

func findThisInstance(cfg *vmwareConfig, instances []Instance) (*Instance, error) {
	for _, instance := range instances {
		if strings.Contains(cfg.Global.VMName, instance.InstanceID) {
			return &Instance{
				InstanceID: instance.InstanceID,
				PrivateIP:  instance.PrivateIP,
			}, nil
		}
	}

	return nil, errors.New("Unable to find VM instance")
}

const (
	roundTripperDefaultCount = 3
)

type vmwareConfig struct {
	Global struct {
		// vCenter username.
		User string `gcfg:"user"`
		// vCenter password in clear text.
		Password string `gcfg:"password"`
		// vCenter IP.
		VCenterIP string `gcfg:"server"`
		// vCenter port.
		VCenterPort string `gcfg:"port"`
		// True if vCenter uses self-signed cert.
		InsecureFlag bool `gcfg:"insecure-flag"`
		// Datacenter in which VMs are located.
		Datacenter string `gcfg:"datacenter"`
		// Datastore in which vmdks are stored.
		Datastore string `gcfg:"datastore"`
		// WorkingDir is path where VMs can be found.
		WorkingDir string `gcfg:"working-dir"`
		// Soap round tripper count (retries = RoundTripper - 1)
		RoundTripperCount uint `gcfg:"soap-roundtrip-count"`
		// VMName is the VM name of virtual machine
		VMName string `gcfg:"vm-name"`
	}

	Disk struct {
		// SCSIControllerType defines SCSI controller to be used.
		SCSIControllerType string `dcfg:"scsicontrollertype"`
	}
}

func readConfig(config io.Reader) (*vmwareConfig, error) {
	if config == nil {
		return nil, errors.New("No VMWare config file given")
	}

	var cfg vmwareConfig
	err := gcfg.ReadInto(&cfg, config)
	return &cfg, err
}

// NewClient creates a govmomi.Client for use in the examples
func NewClient(ctx context.Context, cfg *vmwareConfig) (*govmomi.Client, error) {
	flag.Parse()

	u, err := url.Parse(fmt.Sprintf("https://%s:%s/sdk", cfg.Global.VCenterIP, cfg.Global.VCenterPort))
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(cfg.Global.User, cfg.Global.Password)

	c, err := govmomi.NewClient(ctx, u, cfg.Global.InsecureFlag)
	if err != nil {
		return nil, err
	}

	roundTripperCount := cfg.Global.RoundTripperCount
	if roundTripperCount == 0 {
		roundTripperCount = roundTripperDefaultCount
	}

	c.RoundTripper = vim25.Retry(c.RoundTripper, vim25.TemporaryNetworkError(int(roundTripperCount)))

	return c, nil
}
