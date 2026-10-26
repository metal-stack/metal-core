package frrkeepinterfaces

type FrrKeepInterfaces interface {
	// Remove Interface configuration instead of deleting them outright in FRR
	Enabled() bool
}

// Old behavior kept around for compatibility reasons
type Compat struct{}

func (Compat) Enabled() bool { return false }

// Required for newer FRR versions
// - FRR 10.3 in AsterNOS refuses to apply a frr.conf that deletes active interfaces  (observed on asternos V6.1 based on Community 202511)
type Modern struct{}

func (Modern) Enabled() bool { return true }
