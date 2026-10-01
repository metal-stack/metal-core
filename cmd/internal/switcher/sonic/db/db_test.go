package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/sonic/db/test"
	"github.com/stretchr/testify/require"
)

func TestNewRedisClient_UnixSocket(t *testing.T) {
	mr := miniredis.RunT(t)
	sock := test.StartUnixProxy(t, mr.Addr())

	_, err := newRedisClient(instance{Addr: sock}, 0)
	require.NoError(t, err)
}

func TestNewRedisClient_UnixSocket_Auth(t *testing.T) {
	const password = "s3cret"

	mr := miniredis.RunT(t)
	mr.RequireAuth(password)
	sock := test.StartUnixProxy(t, mr.Addr())

	pwFile := filepath.Join(t.TempDir(), "pw")
	require.NoError(t, os.WriteFile(pwFile, []byte(password+"\n"), 0o600))

	_, err := newRedisClient(instance{Addr: sock, PasswordPath: pwFile}, 0)
	require.NoError(t, err)
}
