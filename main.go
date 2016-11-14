package main

import (
	"flag"

	"io/ioutil"

	log "github.com/Sirupsen/logrus"
	"github.com/sky-uk/etcd-bootstrap/lib"
)

var outputFilename string

func init() {
	const defaultOutputFilename = "/var/run/etcd-bootstrap.conf"

	flag.StringVar(&outputFilename, "o", defaultOutputFilename,
		"location to write environment variables for etcd to use")
}

func main() {
	flag.Parse()

	out := "# created by etcd-bootstrap\n"

	bootstrapper := bootstrap.Default()
	out += bootstrapper.Bootstrap()

	log.Infof("Writing environment variables to %s", outputFilename)
	err := ioutil.WriteFile(outputFilename, []byte(out), 0644)
	if err != nil {
		log.Fatalf("Unable to write to %s: %v", outputFilename, err)
	}
}
