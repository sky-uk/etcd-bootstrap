package cmd

import (
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/bootstrap"
	vmware_provider "github.com/sky-uk/etcd-bootstrap/cloud/vmware"
	"github.com/sky-uk/etcd-bootstrap/etcd"
	"github.com/spf13/cobra"
)

const (
	defaultVMWarePort               = 443
	defaultVMwareInsecureSkipVerify = false
	defaultVMwareAttempts           = 3

	vmwarePasswordEnvironmentVariable = "VSPHERE_PASSWORD"
)

// vmwareCmd represents the generate config command for VMware etcd clusters
var vmwareCmd = &cobra.Command{
	Use:    "vmware",
	Short:  "Generates config for a VMware etcd cluster",
	Run:    vmware,
	PreRun: checkVMwareParams,
}

var (
	vmwareUsername           string
	vmwarePassword           string
	vmwareHost               string
	vmwarePort               uint
	vmwareInsecureSkipVerify bool
	vmwareAttempts           uint
	vmwareVMName             string
	vmwareEnvironment        string
	vmwareRole               string
)

func init() {
	RootCmd.AddCommand(vmwareCmd)

	// vmware flags
	vmwareCmd.Flags().StringVar(&vmwareUsername, "vsphere-username", "",
		"username for vSphere API")
	vmwareCmd.Flags().StringVar(&vmwareHost, "vsphere-host", "",
		"host address for vSphere API")
	vmwareCmd.Flags().UintVar(&vmwarePort, "vsphere-port", defaultVMWarePort,
		"port for vSphere API")
	vmwareCmd.Flags().BoolVar(&vmwareInsecureSkipVerify, "insecure-skip-verify",
		defaultVMwareInsecureSkipVerify, "skip SSL verification when communicating with the vSphere host")
	vmwareCmd.Flags().UintVar(&vmwareAttempts, "max-api-attempts", defaultVMwareAttempts,
		"number of attempts to make against the vSphere SOAP API (in case of temporary failure)")
	vmwareCmd.Flags().StringVar(&vmwareVMName, "vm-name", "",
		"node name in vSphere of this VM")
	vmwareCmd.Flags().StringVar(&vmwareEnvironment, "environment", "",
		"value of the 'tags_environment' extra configuration option in vSphere to filter nodes by")
	vmwareCmd.Flags().StringVar(&vmwareRole, "role", "",
		"value of the 'tags_role' extra configuration option in vSphere to filter nodes by")

	// vmware environment variables
	vmwarePassword = os.Getenv(vmwarePasswordEnvironmentVariable)
}

func vmware(cmd *cobra.Command, args []string) {
	vmwareProvider, err := vmware_provider.NewVMware(&vmware_provider.Config{
		User:              vmwareUsername,
		Password:          vmwarePassword,
		VCenterHost:       vmwareHost,
		VCenterPort:       vmwarePort,
		InsecureFlag:      vmwareInsecureSkipVerify,
		RoundTripperCount: vmwareAttempts,
		VMName:            vmwareVMName,
		Environment:       vmwareEnvironment,
		Role:              vmwareRole,
	})
	if err != nil {
		log.Fatalf("Failed to create VMware provider: %v", err)
	}

	etcdCluster, err := etcd.New(vmwareProvider)
	if err != nil {
		log.Fatalf("Failed to create etcd cluster API: %v", err)
	}
	bootstrapper, err := bootstrap.New(vmwareProvider, vmwareProvider, etcdCluster)
	if err != nil {
		log.Fatalf("Failed to create etcd bootstrapper: %v", err)
	}

	if err := bootstrapper.GenerateEtcdFlagsFile(outputFilename); err != nil {
		log.Fatalf("Failed to generate etcd flags file: %v", err)
	}
}

func checkVMwareParams(cmd *cobra.Command, args []string) {
	checkRequiredFlag(vmwareUsername, "--vsphere-username")
	checkRequiredFlag(vmwareHost, "--vsphere-host")
	checkRequiredFlag(vmwareVMName, "--vm-name")
	checkRequiredFlag(vmwareEnvironment, "--environment")
	checkRequiredFlag(vmwareRole, "--role")
	checkRequiredEnvironmentVariable(vmwarePassword, vmwarePasswordEnvironmentVariable)
}
