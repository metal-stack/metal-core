package templates

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"text/template"

	"github.com/coreos/go-systemd/v22/unit"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

type Reloader func() error

type Applier struct {
	Dest              string
	Reloader          Reloader
	Tmp               string
	Tpl               *template.Template
	ValidationService string
}

func (a *Applier) Apply(c *types.Conf) error {
	err := write(c, a.Tpl, a.Tmp)
	if err != nil {
		return err
	}

	equal, err := areEqual(a.Tmp, a.Dest)
	if err != nil {
		return err
	}
	if equal {
		return os.Remove(a.Tmp)
	}

	err = validate(a.ValidationService, a.Tmp)
	if err != nil {
		return err
	}

	err = os.Rename(a.Tmp, a.Dest)
	if err != nil {
		return err
	}
	return a.Reloader()
}

func write(c *types.Conf, tpl *template.Template, tmpPath string) error {
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	err = tpl.Execute(f, c)
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func areEqual(tmp, dest string) (bool, error) {
	tmpChecksum, err := checksum(tmp)
	if err != nil {
		return false, err
	}

	destChecksum, err := checksum(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return bytes.Equal(tmpChecksum, destChecksum), nil
}

func validate(service string, path string) error {
	u := fmt.Sprintf("%s@%s.service", service, unit.UnitNamePathEscape(path))
	if err := dbus.Start(u); err != nil {
		return fmt.Errorf("%s failed: %w", u, err)
	}
	return nil
}

func checksum(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
