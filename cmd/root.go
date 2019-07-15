package cmd

import (
	"fmt"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type registrationProvider int

const (
	defaultOutputFilename = "/var/run/etcd-bootstrap.conf"
)

func (r registrationProvider) String(types []string) string {
	return types[r]
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "etcd-bootstrap",
	Short: "Bootstrap and register etcd clusters",
}

var (
	debugLogging bool
	// injected by "go tool link -X"
	version string
	// injected by "go tool link -X"
	buildTime string

	outputFileName string
)

func init() {
	cobra.OnInitialize(initLogs)
	RootCmd.Version = fmt.Sprintf("%s (%s)", version, buildTime)
	RootCmd.PersistentFlags().BoolVarP(&debugLogging, "debug", "X", false,
		"enable debug logging")
	RootCmd.PersistentFlags().StringVarP(&outputFileName, "output-file", "o", defaultOutputFilename,
		"location to write environment variables for etcd to use")
}

func initLogs() {
	if debugLogging {
		log.SetLevel(log.DebugLevel)
	}
}

func checkRequiredFlag(value, flagName string) {
	if strings.TrimSpace(value) == "" {
		log.Fatalf("The %s flag is required", flagName)
	}
}

func checkRequiredEnvironmentVariable(value, envName string) {
	if strings.TrimSpace(value) == "" {
		log.Fatalf("The %s environment variable is required", envName)
	}
}
