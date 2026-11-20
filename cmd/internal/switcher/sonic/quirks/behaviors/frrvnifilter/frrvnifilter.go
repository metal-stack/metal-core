package frrvnifilter

type FrrDisableVniFilter interface {
	// Do we need to omit the "match evpn vni" route-maps applied outbound to firewall neighbors?
	// FRR < 9.0.4 does not handle them correctly: https://github.com/FRRouting/frr/issues/14419#issuecomment-1836456246
	Enabled() bool
}

// FRR versions affected by the bug
// - Edgecore SONiC 202211 ships FRR 9.0.1
type Broken struct{}

func (Broken) Enabled() bool { return true }

type Fixed struct{}

func (Fixed) Enabled() bool { return false }
