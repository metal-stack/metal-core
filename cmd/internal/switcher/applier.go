package switcher

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"io"
	"os"
)

type Reloader interface {
	Reload() error
}

type Renderer interface {
	Render(io.Writer, *Conf) error
}

type Validator interface {
	Validate(path string) error
}

type destConfig struct {
	dest     string
	renderer Renderer
	tmpFile  string
}

func newDestConfig(dest string, renderer Renderer) *destConfig {
	return &destConfig{
		dest:     dest,
		renderer: renderer,
		tmpFile:  dest + ".tmp",
	}
}

type networkApplier struct {
	destConfigs []*destConfig
	reloader    Reloader
	validator   Validator
}

// Apply applies the given configuration.
func (n *networkApplier) Apply(c *Conf) error {
	for _, d := range n.destConfigs {
		err := write(c, d)
		if err != nil {
			return err
		}
	}

	for _, d := range n.destConfigs {
		err := n.validator.Validate(d.tmpFile)
		if err != nil {
			return err
		}
	}

	equals := true
	for _, d := range n.destConfigs {
		equals = equals && areEqual(d.tmpFile, d.dest)
	}

	if equals {
		return nil
	}

	for _, d := range n.destConfigs {
		err := os.Rename(d.tmpFile, d.dest)
		if err != nil {
			return err
		}
	}

	return n.reloader.Reload()
}

func write(c *Conf, d *destConfig) error {
	f, err := os.OpenFile(d.tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	err = d.renderer.Render(w, c)
	if err != nil {
		_ = f.Close()
		return err
	}

	err = w.Flush()
	if err != nil {
		_ = f.Close()
		return err
	}

	return f.Close()
}

func areEqual(source, target string) bool {
	sourceChecksum, err := checksum(source)
	if err != nil {
		return false
	}

	targetChecksum, err := checksum(target)
	if err != nil {
		return false
	}

	return bytes.Equal(sourceChecksum, targetChecksum)
}

func checksum(file string) ([]byte, error) {
	f, err := os.Open(file)
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
