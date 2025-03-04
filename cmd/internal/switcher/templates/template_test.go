package templates

import (
	"bytes"
	"log"
	"os"
	"path"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"
)

func TestInterfacesTemplate(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			tpl := InterfacesTemplate("")
			verifyTemplate(t, tpl, &c, path.Join("test_data", tt, "interfaces"))
		})
	}
}

func TestInterfacesTemplateWithDownPorts(t *testing.T) {
	c := readConf(t, path.Join("test_down_interfaces", "conf_with_downports.yaml"))
	c = *c.NewWithoutDownPorts()
	tpl := InterfacesTemplate("")
	verifyTemplate(t, tpl, &c, path.Join("test_down_interfaces", "interfaces_with_downports"))
}

func TestCumulusFrrTemplate(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			err := c.FillRouteMapsAndIPPrefixLists()
			require.NoError(t, err)
			tpl := CumulusFrrTemplate("")
			verifyTemplate(t, tpl, &c, path.Join("test_data", tt, "cumulus_frr.conf"))
		})
	}
}

func TestSonicFrrTpl(t *testing.T) {
	tests := listTestCases()
	for i := range tests {
		tt := tests[i]
		t.Run(tt, func(t *testing.T) {
			c := readConf(t, path.Join("test_data", tt, "conf.yaml"))
			c.CapitalizeVrfName()
			err := c.FillRouteMapsAndIPPrefixLists()
			require.NoError(t, err)
			tpl := SonicFrrTemplate("")
			verifyTemplate(t, tpl, &c, path.Join("test_data", tt, "sonic_frr.conf"))
		})
	}
}

func TestCustomInterfacesTemplate(t *testing.T) {
	c := readConf(t, "test_data/dev/conf.yaml")
	tpl := InterfacesTemplate("test_data/dev/customtpl/interfaces.tpl")
	verifyTemplate(t, tpl, &c, "test_data/dev/customtpl/interfaces")
}

func TestCustomCumulusFrrTemplate(t *testing.T) {
	c := readConf(t, "test_data/dev/conf.yaml")
	err := c.FillRouteMapsAndIPPrefixLists()
	require.NoError(t, err)
	tpl := CumulusFrrTemplate("test_data/dev/customtpl/frr.tpl")
	verifyTemplate(t, tpl, &c, "test_data/dev/customtpl/frr.conf")
}

func TestCustomSonicFrrTemplate(t *testing.T) {
	c := readConf(t, "test_data/dev/conf.yaml")
	err := c.FillRouteMapsAndIPPrefixLists()
	require.NoError(t, err)
	tpl := SonicFrrTemplate("test_data/dev/customtpl/frr.tpl")
	verifyTemplate(t, tpl, &c, "test_data/dev/customtpl/frr.conf")
}

func verifyTemplate(t *testing.T, tpl *template.Template, c *types.Conf, expectedFilename string) {
	actual := renderToString(t, tpl, c)
	expected := readExpected(t, expectedFilename)
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("%s render differs:%s", expectedFilename, diff)
	}
}

func renderToString(t *testing.T, tpl *template.Template, c *types.Conf) string {
	var b bytes.Buffer
	err := tpl.Execute(&b, c)
	require.NoError(t, err, "Couldn't render configuration")
	return b.String()
}

func readConf(t *testing.T, i string) types.Conf {
	c := types.Conf{}
	b, err := os.ReadFile(i)
	require.NoError(t, err, "unexpected error when reading testing input")

	err = yaml.Unmarshal(b, &c) // nolint:musttag
	require.NoError(t, err, "unexpected error when unmarshalling testing input")
	return c
}

func readExpected(t *testing.T, e string) string {
	ex, err := os.ReadFile(e)
	require.NoError(t, err, "Couldn't read %s", e)
	return string(ex)
}

func listTestCases() []string {
	files, err := os.ReadDir("test_data")
	if err != nil {
		log.Fatal(err)
	}

	r := []string{}
	for _, f := range files {
		if f.IsDir() {
			r = append(r, f.Name())
		}
	}
	return r
}
