package vmware

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/sky-uk/etcd-bootstrap/lib/cloud"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type vmwareMembers struct {
	instances []cloud.Instance
	instance  cloud.Instance
}

func (m vmwareMembers) GetInstances() []cloud.Instance {
	return m.instances
}

func (m vmwareMembers) GetLocalInstance() cloud.Instance {
	return m.instance
}

func (m vmwareMembers) UpdateDNS(name string) error {
	// No DNS provider is enabled for VMWare
	return nil
}

// NewVMware returns the Members this local instance belongs to.
func NewVMware(cfg *Config) (cloud.Cloud, error) {
	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	c, err := NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer c.Logout(ctx)

	instances, err := findAllInstances(ctx, c, cfg.Environment, cfg.Role)
	if err != nil {
		return nil, err
	}

	instance, err := findThisInstance(cfg, instances)
	if err != nil {
		return nil, err
	}

	members := vmwareMembers{
		instances: instances,
		instance:  *instance,
	}

	return members, nil
}

func findAllInstances(ctx context.Context, c *govmomi.Client, env, role string) ([]cloud.Instance, error) {
	m := view.NewManager(c.Client)

	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}

	defer v.Destroy(ctx)

	// Reference: http://pubs.vmware.com/vsphere-60/topic/com.vmware.wssdk.apiref.doc/vim.VirtualMachine.html
	var vms []mo.VirtualMachine

	// Does restricting the scope for the fields we're after make it faster?
	err = v.Retrieve(ctx, []string{"VirtualMachine"}, []string{"config.name", "config.extraConfig", "summary.runtime", "summary.guest"}, &vms)
	if err != nil {
		return nil, err
	}

	var instances []cloud.Instance

	var matched []mo.VirtualMachine
	for _, vm := range vms {
		if matchesTag(vm, "tags_environment", env) && matchesTag(vm, "tags_role", role) {
			matched = append(matched, vm)
		}
	}

	for _, vm := range matched {
		if vm.Summary.Runtime.PowerState == types.VirtualMachinePowerStatePoweredOn {
			instances = append(instances, cloud.Instance{
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

func findThisInstance(cfg *Config, instances []cloud.Instance) (*cloud.Instance, error) {
	for _, instance := range instances {
		if strings.Contains(cfg.VMName, instance.InstanceID) {
			return &cloud.Instance{
				InstanceID: instance.InstanceID,
				PrivateIP:  instance.PrivateIP,
			}, nil
		}
	}

	return nil, errors.New("Unable to find VM instance")
}

// Config is the configuration required to talk to the vSphere API to fetch a list of nodes
type Config struct {
	// vCenter username.
	User string
	// vCenter password in clear text.
	Password string
	// vCenter Hostname or IP.
	VCenterHost string
	// vCenter port.
	VCenterPort string
	// True if vCenter uses self-signed cert.
	InsecureFlag bool
	// Soap round tripper count (retries = RoundTripper - 1)
	RoundTripperCount uint
	// VMName is the VM name of virtual machine
	VMName string
	// Environment tag to filter by
	Environment string
	// Role tag to filter by
	Role string
}

// NewClient creates a govmomi.Client for use in the examples
func NewClient(ctx context.Context, cfg *Config) (*govmomi.Client, error) {
	flag.Parse()

	u, err := url.Parse(fmt.Sprintf("https://%s:%s/sdk", cfg.VCenterHost, cfg.VCenterPort))
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(cfg.User, cfg.Password)

	c, err := govmomi.NewClient(ctx, u, cfg.InsecureFlag)
	if err != nil {
		return nil, err
	}

	c.RoundTripper = vim25.Retry(c.RoundTripper, vim25.TemporaryNetworkError(int(cfg.RoundTripperCount)))

	return c, nil
}
