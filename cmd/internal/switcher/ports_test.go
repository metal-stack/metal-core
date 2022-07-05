package switcher

import (
	"reflect"
	"testing"

	"github.com/go-redis/redis/v8"
	"github.com/go-redis/redismock/v8"
)

func Test_applyMtus(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: "|",
	}
	given := map[string]*port{
		"non-existing": {mtu: "9000"},
		"changed":      {mtu: "new mtu"},
		"unchanged":    {mtu: "unchanged"},
	}

	mock.MatchExpectationsInOrder(false)

	mock.ExpectHGet("PORT|non-existing", "mtu").SetErr(redis.Nil)
	mock.ExpectHSet("PORT|non-existing", "mtu", "9000").SetVal(1)

	mock.ExpectHGet("PORT|changed", "mtu").SetVal("old mtu")
	mock.ExpectHSet("PORT|changed", "mtu", "new mtu").SetVal(1)

	mock.ExpectHGet("PORT|unchanged", "mtu").SetVal("unchanged")

	if err := applyMtus(db, given); err != nil {
		t.Errorf("applyMtus() unexpected error %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func Test_applyPorts(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	db := &ConfigDB{
		rdb:       rdb,
		separator: "|",
	}
	given := map[string]*port{
		"non-vrf":      {},
		"non-existing": {vrfName: "vrf1"},
		"changed":      {vrfName: "new vrf"},
		"unchanged":    {vrfName: "unchanged"},
	}

	mock.MatchExpectationsInOrder(false)

	mock.ExpectKeys("INTERFACE|*").SetVal([]string{"INTERFACE|changed", "INTERFACE|unchanged", "leftover"})

	mock.ExpectHSet("INTERFACE|non-existing", "vrf_name", "vrf1").SetVal(1)

	mock.ExpectHGetAll("INTERFACE|changed").SetVal(map[string]string{"vrf_name": "old vrf"})
	mock.ExpectHSet("INTERFACE|changed", "vrf_name", "new vrf").SetVal(1)

	mock.ExpectHGetAll("INTERFACE|unchanged").SetVal(map[string]string{"vrf_name": "unchanged"})

	//mock.ExpectDel("leftover").SetVal(1)

	if err := applyPorts(db, given); err != nil {
		t.Errorf("applyPorts() unexpected error %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func Test_getPorts(t *testing.T) {
	given := &Conf{Ports: Ports{
		Underlay:      []string{"underlay"},
		Unprovisioned: []string{"unprovisioned"},
		Vrfs:          map[string]*Vrf{"vrf": {Neighbors: []string{"vrf neighbor"}}},
		Firewalls:     map[string]*Firewall{"firewall": {Port: "firewall port"}},
	}}

	want := map[string]*port{
		"underlay":      {mtu: "9216"},
		"unprovisioned": {mtu: "9000"},
		"vrf neighbor":  {mtu: "9000", vrfName: "vrf"},
		"firewall port": {mtu: "9216"},
	}

	if got := getPorts(given); !reflect.DeepEqual(got, want) {
		t.Errorf("getPorts() = %v, want %v", got, want)
	}
}
