package templates

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
	"io"
	"os"
	"text/template"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/metal-stack/metal-core/cmd/internal/dbus"
)

func write(c *types.Conf, tpl *template.Template, tmpPath string) error {
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tpl.Execute(f, c)
	if err != nil {
		return err
	}

	return nil
}

func validate(service string, path string) error {
	u := fmt.Sprintf("%s@%s.service", service, unit.UnitNamePathEscape(path))
	if err := dbus.Start(u); err != nil {
		return fmt.Errorf("%s@%s: %w", service, unit.UnitNamePathEscape(path), err)
	}
	return nil
}

func move(src, dest string) (bool, error) {
	sourceChecksum, err := checksum(src)
	if err != nil {
		return false, err
	}

	targetChecksum, err := checksum(dest)
	if err != nil {
		return false, err
	}

	if bytes.Equal(sourceChecksum, targetChecksum) {
		return false, os.Remove(src)
	}
	return true, os.Rename(src, dest)
}

func checksum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
