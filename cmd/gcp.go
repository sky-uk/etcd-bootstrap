package cmd

import (
	log "github.com/sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/bootstrap"
	gcp_provider "github.com/sky-uk/etcd-bootstrap/cloud/gcp"
	"github.com/sky-uk/etcd-bootstrap/etcd"
	"github.com/spf13/cobra"
)

// gcpCmd represents the generate config command for GCP etcd clusters
var gcpCmd = &cobra.Command{
	Use:    "gcp",
	Short:  "Generates config for a GCP etcd cluster",
	Run:    gcp,
	PreRun: checkGCPParams,
}

var (
	gcpProjectID   string
	gcpEnvironment string
	gcpRole        string
)

func init() {
	RootCmd.AddCommand(gcpCmd)

	gcpCmd.Flags().StringVar(&gcpProjectID, "project-id", "",
		"value of the GCP 'project id' to query")
	gcpCmd.Flags().StringVar(&gcpEnvironment, "environment", "",
		"value of the 'environment' label in GCP nodes to filter them by")
	gcpCmd.Flags().StringVar(&gcpRole, "role", "",
		"value of the 'role' label in GCP nodes to filter them by")
}

func gcp(cmd *cobra.Command, args []string) {
	gcpProvider, err := gcp_provider.NewGCP(&gcp_provider.Config{
		ProjectID:   gcpProjectID,
		Environment: gcpEnvironment,
		Role:        gcpRole,
	})
	if err != nil {
		log.Fatalf("Failed to create GCP provider: %v", err)
	}

	etcdCluster, err := etcd.New(gcpProvider)
	if err != nil {
		log.Fatalf("Failed to create etcd cluster API: %v", err)
	}
	bootstrapper, err := bootstrap.New(gcpProvider, etcdCluster)
	if err != nil {
		log.Fatalf("Failed to create etcd bootstrapper: %v", err)
	}

	if err := bootstrapper.GenerateEtcdFlagsFile(outputFilename); err != nil {
		log.Fatalf("Failed to generate etcd flags file: %v", err)
	}
}

func checkGCPParams(cmd *cobra.Command, args []string) {
	checkRequiredFlag(gcpProjectID, "--project-id")
	checkRequiredFlag(gcpEnvironment, "--environment")
	checkRequiredFlag(gcpRole, "--role")
}
