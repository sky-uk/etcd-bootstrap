package bootstrap

func (b *bootstrapper) BootstrapRoute53(zoneID, name string) error {
	var ips []string
	for _, instance := range b.asg.GetInstances() {
		ips = append(ips, instance.PrivateIP)
	}
	return b.r53.UpdateARecords(zoneID, name, ips)
}
