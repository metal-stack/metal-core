package switcher

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"git.f-i-ts.de/cloud-native/metallib/vlan"
	"github.com/pkg/errors"
)

// FillVLANIDs fills the given configuration object with switch-local VLAN-IDs
// if they are present in the given VLAN-Mapping
// otherwise: new available VLAN-IDs will be used
func (c *Conf) FillVLANIDs(m vlan.Mapping) error {
Tloop:
	for _, t := range c.Tenants {
		for vlan, vni := range m {
			if vni == t.VNI {
				t.VLANID = vlan
				continue Tloop
			}
		}
		vlanids, err := m.ReserveVlanIDs(1)
		if err != nil {
			return err
		}
		vlan := vlanids[0]
		t.VLANID = vlan
		m[vlan] = t.VNI
	}
	return nil
}

func (c *Conf) validate() error {
	tmpfile, err := ioutil.TempFile("", "frr.conf")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	frr := NewFrrApplier(c)
	err = c._write(tmpfile, frr)
	if err != nil {
		return errors.Wrap(err, "could not write frr.conf for validation")
	}

	f, err := filepath.Abs(tmpfile.Name())
	if err != nil {
		return errors.Wrap(err, "could not find absolute path")
	}
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command("vtysh", "-C", "-f", f)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("validation returned error; stdout: %v, stderr: %v", outbuf.String(), errbuf.String()))
	}
	return nil
}

func (c *Conf) _write(f *os.File, a Applier) error {
	w := bufio.NewWriter(f)
	err := a.Render(w)
	if err != nil {
		return err
	}
	err = w.Flush()
	if err != nil {
		return err
	}
	return nil
}

func (c *Conf) applyFrr() error {
	f, err := os.OpenFile("/etc/frr/frr.conf", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	frr := NewFrrApplier(c)
	err = c._write(f, frr)
	if err != nil {
		return err
	}

	err = frr.Reload()
	if err != nil {
		return err
	}

	return nil
}

func (c *Conf) applyInterfaces() error {
	f, err := os.OpenFile("/etc/network/interfaces", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	frr := NewInterfacesApplier(c)
	err = frr.Render(w)
	if err != nil {
		return err
	}

	err = w.Flush()
	if err != nil {
		return err
	}

	err = frr.Reload()
	if err != nil {
		return err
	}

	return nil
}

// Apply applies the configuration to the switch
func (c *Conf) Apply() error {
	err := c.applyInterfaces()
	if err != nil {
		return err
	}

	err = c.applyFrr()
	if err != nil {
		return err
	}
	return nil
}
