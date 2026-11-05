package bisdn

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/vishvananda/netlink"

	"github.com/metal-stack/metal-core/cmd/internal/dbus"
	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

const (
	networkdDir = "/etc/systemd/network"

	// fileMarker starts a file in the rendered networkd template; the rest of
	// the line is the file name.
	fileMarker = "#> "
)

// managedFile matches the networkd files that belong to metal-core. Other files
// in the directory, such as the mgmt VRF and eth0, are left alone.
var managedFile = regexp.MustCompile(`^[0-9]{2}-bisdn-[A-Za-z0-9._-]+\.(netdev|network)$`)

type (
	// links changes kernel netdevs where networkd can't: it never deletes a
	// netdev or releases a port from its master when a file goes away.
	links interface {
		Delete(name string) error
		Release(name string) error
	}

	networkdApplier struct {
		dir     string
		tpl     *template.Template
		links   links
		reload  func(ctx context.Context) error
		log     *slog.Logger
		applied bool
	}

	netlinkLinks struct{}
)

func newNetworkdApplier(log *slog.Logger, tpl *template.Template) *networkdApplier {
	return &networkdApplier{
		dir:    networkdDir,
		tpl:    tpl,
		links:  netlinkLinks{},
		reload: dbus.ReloadNetworkd,
		log:    log,
	}
}

// Apply renders the networkd files, removes kernel state that the new files no
// longer describe, installs the files and reloads systemd-networkd.
func (a *networkdApplier) Apply(ctx context.Context, cfg *types.Conf) error {
	var b bytes.Buffer
	if err := a.tpl.Execute(&b, cfg); err != nil {
		return fmt.Errorf("failed to render networkd files: %w", err)
	}
	desired, err := splitFiles(b.Bytes())
	if err != nil {
		return err
	}

	current, err := readManagedFiles(a.dir)
	if err != nil {
		return err
	}

	changed := !maps.EqualFunc(current, desired, bytes.Equal)
	if !changed && a.applied {
		a.log.Debug("networkd files are unchanged, nothing to do")
		return nil
	}
	a.applied = false

	if changed {
		// Clean up before the files are replaced: if a step fails, the old
		// files are still there and the next run computes the same changes.
		if err := a.cleanup(current, desired); err != nil {
			return err
		}
		if err := writeFiles(a.dir, current, desired); err != nil {
			return err
		}
	}

	a.log.Debug("reload systemd-networkd")
	if err := a.reload(ctx); err != nil {
		return err
	}
	a.applied = true
	return nil
}

// cleanup deletes netdevs whose .netdev file is removed or changed (networkd
// creates the changed ones again on reload) and releases links that no
// .network file matches anymore from their bridge or VRF.
func (a *networkdApplier) cleanup(current, desired map[string][]byte) error {
	var errs []error

	// netdevs that metal-core creates are never released, only deleted
	keepLinks := map[string]bool{}
	for name, content := range current {
		if strings.HasSuffix(name, ".netdev") {
			keepLinks[iniValue(content, "NetDev", "Name")] = true
		}
	}
	for name, content := range desired {
		if strings.HasSuffix(name, ".network") {
			keepLinks[iniValue(content, "Match", "Name")] = true
		}
	}

	for _, name := range slices.Sorted(maps.Keys(current)) {
		content := current[name]
		switch {
		case strings.HasSuffix(name, ".netdev"):
			if bytes.Equal(content, desired[name]) {
				continue
			}
			netdev := iniValue(content, "NetDev", "Name")
			if netdev == "" {
				continue
			}
			a.log.Info("delete netdev", "netdev", netdev, "file", name)
			if err := a.links.Delete(netdev); err != nil {
				errs = append(errs, fmt.Errorf("failed to delete netdev %s: %w", netdev, err))
			}
		case strings.HasSuffix(name, ".network"):
			link := iniValue(content, "Match", "Name")
			if link == "" || keepLinks[link] {
				continue
			}
			a.log.Info("release link", "link", link, "file", name)
			if err := a.links.Release(link); err != nil {
				errs = append(errs, fmt.Errorf("failed to release link %s: %w", link, err))
			}
		}
	}

	return errors.Join(errs...)
}

// splitFiles splits the rendered template into files, each starting with a
// fileMarker line.
func splitFiles(rendered []byte) (map[string][]byte, error) {
	files := map[string][]byte{}
	var (
		name    string
		content []string
	)
	flush := func() {
		if name != "" {
			files[name] = []byte(strings.TrimSpace(strings.Join(content, "\n")) + "\n")
		}
	}

	for line := range strings.Lines(string(rendered)) {
		line = strings.TrimRight(line, "\n")
		after, found := strings.CutPrefix(line, fileMarker)
		if !found {
			if name == "" && strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("networkd template: content before the first %q line", fileMarker)
			}
			content = append(content, line)
			continue
		}
		flush()
		name = strings.TrimSpace(after)
		content = nil
		if !managedFile.MatchString(name) {
			return nil, fmt.Errorf("networkd template: invalid file name %q", name)
		}
		if _, ok := files[name]; ok {
			return nil, fmt.Errorf("networkd template: file %q rendered twice", name)
		}
	}
	flush()

	return files, nil
}

func readManagedFiles(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	files := map[string][]byte{}
	for _, e := range entries {
		if !e.Type().IsRegular() || !managedFile.MatchString(e.Name()) {
			continue
		}
		content, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		files[e.Name()] = content
	}
	return files, nil
}

func writeFiles(dir string, current, desired map[string][]byte) error {
	for name, content := range desired {
		if bytes.Equal(current[name], content) {
			continue
		}
		path := filepath.Join(dir, name)
		tmp := filepath.Join(dir, "."+name+".tmp")
		if err := os.WriteFile(tmp, content, 0644); err != nil { // nolint:gosec
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			return err
		}
	}
	for name := range current {
		if _, ok := desired[name]; ok {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// iniValue returns the first value of key in section, or "" if there is none.
func iniValue(content []byte, section, key string) string {
	var inSection bool
	s := bufio.NewScanner(bytes.NewReader(content))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if strings.HasPrefix(line, "[") {
			inSection = line == "["+section+"]"
			continue
		}
		if !inSection {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if found && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func (netlinkLinks) Delete(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return err
	}
	return netlink.LinkDel(link)
}

func (netlinkLinks) Release(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		if errors.As(err, &netlink.LinkNotFoundError{}) {
			return nil
		}
		return err
	}
	if link.Attrs().MasterIndex == 0 {
		return nil
	}
	return netlink.LinkSetNoMaster(link)
}
