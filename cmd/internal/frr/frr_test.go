package frr

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestGetBGPStates_Cumulus parses a real `show bgp vrf all neighbors json` dump captured from a Cumulus switch.
func TestGetBGPStates_Cumulus(t *testing.T) {
	file := "testfiles/bgp-neighbors-cumulus.json"
	touchNow(t, file)

	states, err := GetBGPStates(file)
	if err != nil {
		t.Fatalf("GetBGPStates returned error: %v", err)
	}

	// 19 neighbours across all VRFs (vrfId/vrfName keys are skipped).
	if len(states) != 19 {
		t.Fatalf("expected 19 BGP port states, got %d", len(states))
	}

	// swp31 – FABRIC spine peer in the default VRF.
	swp31 := states["swp31"]
	assertStringField(t, "swp31 Neighbor", swp31.Neighbor, "fra-equ01-spine02")
	assertStringField(t, "swp31 PeerGroup", swp31.PeerGroup, "FABRIC")
	assertStringField(t, "swp31 BgpState", swp31.BgpState, "Established")
	assertStringField(t, "swp31 VrfName", swp31.VrfName, "default")
	assertInt64Field(t, "swp31 BgpTimerUpEstablished", swp31.BgpTimerUpEstablished, 1737530383)
	assertInt64Field(t, "swp31 SentPrefixCounter", swp31.SentPrefixCounter, 0)          // not present in Cumulus output
	assertInt64Field(t, "swp31 AcceptedPrefixCounter", swp31.AcceptedPrefixCounter, 50) // IPv4 Unicast only

	// swp6s0 – FIREWALL peer (IPv4 + IPv6 address families).
	swp6s0 := states["swp6s0"]
	assertStringField(t, "swp6s0 Neighbor", swp6s0.Neighbor, "shoot--pcfgbt--inttest20-firewall-9f9ac")
	assertStringField(t, "swp6s0 PeerGroup", swp6s0.PeerGroup, "FIREWALL")
	assertStringField(t, "swp6s0 BgpState", swp6s0.BgpState, "Established")
	assertInt64Field(t, "swp6s0 AcceptedPrefixCounter", swp6s0.AcceptedPrefixCounter, 1) // IPv4 Unicast=1, IPv6 Unicast=0
}

// TestGetBGPStates_Sonic parses a real dump from a SONiC switch.
func TestGetBGPStates_Sonic(t *testing.T) {
	file := "testfiles/bgp-neighbors-sonic.json"
	touchNow(t, file)

	states, err := GetBGPStates(file)
	if err != nil {
		t.Fatalf("GetBGPStates returned error: %v", err)
	}

	// 19 neighbours across all VRFs.
	if len(states) != 19 {
		t.Fatalf("expected 19 BGP port states, got %d", len(states))
	}

	// Ethernet120 – FABRIC spine peer in the default VRF.
	e120 := states["Ethernet120"]
	assertStringField(t, "Ethernet120 Neighbor", e120.Neighbor, "fra-equ01-spine02")
	assertStringField(t, "Ethernet120 PeerGroup", e120.PeerGroup, "FABRIC")
	assertStringField(t, "Ethernet120 BgpState", e120.BgpState, "Established")
	assertStringField(t, "Ethernet120 VrfName", e120.VrfName, "default")
	assertInt64Field(t, "Ethernet120 BgpTimerUpEstablished", e120.BgpTimerUpEstablished, 1773056572)
	assertInt64Field(t, "Ethernet120 SentPrefixCounter", e120.SentPrefixCounter, 59)         // ipv4Unicast only; l2VpnEvpn not parsed
	assertInt64Field(t, "Ethernet120 AcceptedPrefixCounter", e120.AcceptedPrefixCounter, 95) // ipv4Unicast only

	// Ethernet20 – FIREWALL peer with IPv4 + IPv6 address families.
	e20 := states["Ethernet20"]
	assertStringField(t, "Ethernet20 Neighbor", e20.Neighbor, "shoot--pcfgbt--inttest20-firewall-9f9ac")
	assertStringField(t, "Ethernet20 PeerGroup", e20.PeerGroup, "FIREWALL")
	assertStringField(t, "Ethernet20 BgpState", e20.BgpState, "Established")
	assertInt64Field(t, "Ethernet20 SentPrefixCounter", e20.SentPrefixCounter, 60)        // ipv4Unicast=59 + ipv6Unicast=1
	assertInt64Field(t, "Ethernet20 AcceptedPrefixCounter", e20.AcceptedPrefixCounter, 1) // ipv4Unicast=1, ipv6Unicast=0

	// Ethernet23 – peer in Idle state with no hostname reported.
	e23 := states["Ethernet23"]
	assertStringField(t, "Ethernet23 BgpState", e23.BgpState, "Idle")
	assertStringField(t, "Ethernet23 Neighbor", e23.Neighbor, "") // hostname absent in JSON
	assertInt64Field(t, "Ethernet23 BgpTimerUpEstablished", e23.BgpTimerUpEstablished, 0)
}

func TestGetBGPStates_FileNotFound(t *testing.T) {
	_, err := GetBGPStates("/nonexistent/path/bgp.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestGetBGPStates_FileTooOld(t *testing.T) {
	path := writeTempFRRFile(t, map[string]interface{}{})

	// Back-date the file's modification time by more than one hour.
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatalf("setting file mtime: %v", err)
	}

	_, err := GetBGPStates(path)
	if err == nil {
		t.Fatal("expected error for stale file, got nil")
	}
}

func TestGetBGPStates_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "frr-bad-*.json")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	_, _ = f.WriteString("not valid json")
	_ = f.Close()

	_, err = GetBGPStates(f.Name())
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

// --- helpers ---

func assertStringField(t *testing.T, label string, got *string, want string) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: got nil, want %q", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s: got %q, want %q", label, *got, want)
	}
}

func assertInt64Field(t *testing.T, label string, got *int64, want int64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s: got nil, want %d", label, want)
		return
	}
	if *got != want {
		t.Errorf("%s: got %d, want %d", label, *got, want)
	}
}

// touchNow updates a file's mtime to now so that GetBGPStates does not reject it
// as too old (> 1 h).
func touchNow(t *testing.T, file string) {
	t.Helper()
	now := time.Now()
	if err := os.Chtimes(file, now, now); err != nil {
		t.Fatalf("touching %s: %v", file, err)
	}
}

func writeTempFRRFile(t *testing.T, data interface{}) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "frr-bgp-*.json")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	if err := json.NewEncoder(f).Encode(data); err != nil {
		t.Fatalf("encoding JSON: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing temp file: %v", err)
	}
	return f.Name()
}
