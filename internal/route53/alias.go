package route53

type Aliases struct {
	Target      string
	Entries     []string
	Ipv6Enabled bool
}

func newAliases(target string, entries []string, ipv6Enabled bool) {

}
