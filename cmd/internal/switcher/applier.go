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

type networkApplier struct {
	dest      string
	reloader  Reloader
	renderer  Renderer
	tmpFile   string
	validator Validator
}

// Apply applies the given configuration.
func (n *networkApplier) Apply(c *Conf) error {
	f, err := os.OpenFile(n.tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	w := bufio.NewWriter(f)
	err = n.renderer.Render(w, c)
	if err != nil {
		_ = f.Close()
		return err
	}

	err = w.Flush()
	if err != nil {
		_ = f.Close()
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	err = n.validator.Validate(n.tmpFile)
	if err != nil {
		return err
	}

	equal, err := areEqual(n.tmpFile, n.dest)
	if err != nil || equal {
		return err
	}

	err = os.Rename(n.tmpFile, n.dest)
	if err != nil {
		return err
	}

	return n.reloader.Reload()
}

func areEqual(source, target string) (bool, error) {
	sourceChecksum, err := checksum(source)
	if err != nil {
		return false, err
	}

	targetChecksum, err := checksum(target)
	if err != nil {
		return false, err
	}

	return bytes.Equal(sourceChecksum, targetChecksum), nil
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
