package test

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
	"github.com/valkey-io/valkey-go"
)

func StartValkey(t testing.TB) valkey.Client {
	mr := miniredis.RunT(t)
	// rc := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	vc, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{mr.Addr()},
		// This is required because otherwise we get:
		// unknown subcommand 'TRACKING'. Try CLIENT HELP.: [CLIENT TRACKING ON OPTIN]
		// ClientOption.DisableCache must be true for valkey not supporting client-side caching or not supporting RESP3
		DisableCache: true,
	})
	require.NoError(t, err)
	return vc
}

// CONTINUE:
//
// - add function to load JSON into redis
// - add assert func
