package db

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

// Proxy is necessary because miniredis does not offer a unix socket directly
func startUnixProxy(t *testing.T, tcpAddr string) string {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "redis.sock")
	l, err := net.Listen("unix", sock)
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })

	go func() {
		for {
			client, err := l.Accept()
			if err != nil {
				return
			}
			go proxyConn(client, tcpAddr)
		}
	}()
	return sock
}

func proxyConn(client net.Conn, tcpAddr string) {
	defer func() { _ = client.Close() }()
	server, err := net.Dial("tcp", tcpAddr)
	if err != nil {
		return
	}
	defer func() { _ = server.Close() }()

	go func() { _, _ = io.Copy(server, client) }()
	_, _ = io.Copy(client, server)
}

func TestNewRedisClient_UnixSocket(t *testing.T) {
	mr := miniredis.RunT(t)
	sock := startUnixProxy(t, mr.Addr())

	_, err := newRedisClient(instance{Addr: sock}, 0)
	require.NoError(t, err)
}

func TestNewRedisClient_UnixSocket_Auth(t *testing.T) {
	const password = "s3cret"

	mr := miniredis.RunT(t)
	mr.RequireAuth(password)
	sock := startUnixProxy(t, mr.Addr())

	pwFile := filepath.Join(t.TempDir(), "pw")
	require.NoError(t, os.WriteFile(pwFile, []byte(password+"\n"), 0o600))

	_, err := newRedisClient(instance{Addr: sock, PasswordPath: pwFile}, 0)
	require.NoError(t, err)
}
