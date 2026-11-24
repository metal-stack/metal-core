// Package sonictest provides SONiC databases for tests.
package sonictest

import (
	"log/slog"
	"maps"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
)

// Data is the content of the databases, keyed by their name, e.g. CONFIG_DB.
type Data map[string]test.StringMap

// Databases are the SONiC databases in one miniredis.
type Databases struct {
	Config *db.Config
	mr     *miniredis.Miniredis
}

// NewDatabases starts a miniredis with the SONiC databases and loads the data into them.
func NewDatabases(t testing.TB, data Data) *Databases {
	t.Helper()

	mr := miniredis.RunT(t)
	d := &Databases{
		mr: mr,
		Config: &db.Config{
			Databases: map[string]db.Database{
				"APPL_DB":     {Id: 0, Instance: "redis", Separator: ":"},
				"ASIC_DB":     {Id: 1, Instance: "redis", Separator: ":"},
				"COUNTERS_DB": {Id: 2, Instance: "redis", Separator: ":"},
				"CONFIG_DB":   {Id: 4, Instance: "redis", Separator: "|"},
				"STATE_DB":    {Id: 6, Instance: "redis", Separator: "|"},
			},
			Instances: map[string]db.Instance{
				"redis": {Hostname: mr.Addr()},
			},
		},
	}

	for name, content := range data {
		vc, sep := d.Client(t, name)
		require.NoError(t, test.LoadData(t.Context(), vc, content, sep))
	}
	return d
}

// DB connects to the databases.
func (d *Databases) DB(t testing.TB) *db.DB {
	t.Helper()

	sonicDb, err := db.New(d.Config, slog.Default(), db.SchemeRedis)
	require.NoError(t, err)
	return sonicDb
}

// Client returns a client of the database and its separator.
func (d *Databases) Client(t testing.TB, name string) (valkey.Client, string) {
	t.Helper()

	database, ok := d.Config.Databases[name]
	require.True(t, ok, "unknown database %s", name)

	vc, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:  []string{d.mr.Addr()},
		DisableCache: true,
		SelectDB:     database.Id,
	})
	require.NoError(t, err)
	t.Cleanup(vc.Close)
	return vc, database.Separator
}

// Get returns the content of the database.
func (d *Databases) Get(t testing.TB, name string) test.HashMap {
	t.Helper()

	vc, sep := d.Client(t, name)
	data, err := test.GetData(t.Context(), vc, sep)
	require.NoError(t, err)
	return data
}

// Kernel is a kernel whose netdevs have the admin status of the map, by name.
type Kernel map[string]bool

func (k Kernel) AdminUp() (map[string]bool, error) {
	return maps.Clone(k), nil
}
