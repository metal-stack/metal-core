package switcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"testing"
	"unsafe"

	"github.com/go-redis/redis/v8"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestConfigDBApplier_Apply(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			rdb, r := newRedisMock()

			a := &ConfigDBApplier{
				db: &ConfigDB{rdb: rdb, separator: "|"},
			}
			cfg := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			if err := a.Apply(&cfg); err != nil {
				t.Errorf("Apply() unexpected error = %v", err)
				return
			}
			want := readWanted(t, path.Join("test_data", tt, "configdb.json"))
			if !reflect.DeepEqual(r.records, want) {
				t.Error(cmp.Diff(r.records, want))
			}
		})
	}
}

func readWanted(t *testing.T, p string) map[string][]string {
	data, err := os.ReadFile(p)
	require.NoError(t, err, "Couldn't read %s", p)

	m := make(map[string][]string)
	err = json.Unmarshal(data, &m)
	require.NoError(t, err, "Couldn't unmarshall %s", p)
	return m
}

func newRedisMock() (*redis.Client, *recorder) {
	r := &recorder{make(map[string][]string)}
	// Set MaxRetries to -2 to avoid executing commands
	opt := &redis.Options{MaxRetries: -2}
	client := redis.NewClient(opt)
	client.AddHook(mockHook{r})
	return client, r
}

type recorder struct {
	records map[string][]string
}

func (c recorder) record(cmd redis.Cmder) {
	if cmd.Args()[0] == "hset" {
		key := fmt.Sprint(cmd.Args()[1])
		if _, ok := c.records[key]; ok {
			panic("duplicate hset")
		}
		v := make([]string, len(cmd.Args()[2:]))
		for i, arg := range cmd.Args()[2:] {
			v[i] = fmt.Sprint(arg)
		}
		c.records[key] = v
	}
}

type mockHook struct {
	capture *recorder
}

func (h mockHook) BeforeProcess(ctx context.Context, cmd redis.Cmder) (context.Context, error) {
	h.capture.record(cmd)
	process(cmd)
	return ctx, nil
}

func (h mockHook) AfterProcess(_ context.Context, _ redis.Cmder) error {
	return nil
}

func (h mockHook) BeforeProcessPipeline(_ context.Context, _ []redis.Cmder) (context.Context, error) {
	panic("should not be called")
}

func (h mockHook) AfterProcessPipeline(_ context.Context, _ []redis.Cmder) error {
	panic("should not be called")
}

// Inspirited from inflow in redismock/expect.go
func process(cmd redis.Cmder) {
	v := reflect.ValueOf(cmd).Elem().FieldByName("val")
	// Avoid panic: Value.Set using value obtained using unexported field
	v = reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem()
	v.Set(reflect.ValueOf(determineValue(cmd)))
}

func determineValue(cmd redis.Cmder) interface{} {
	switch cmd.(type) {
	case *redis.IntCmd:
		return int64(1)
	case *redis.StringCmd:
		return ""
	case *redis.StringSliceCmd:
		return make([]string, 0)
	case *redis.StringStringMapCmd:
		return make(map[string]string)
	default:
		panic(fmt.Sprintf("not expected %v", reflect.TypeOf(cmd)))
	}
}
