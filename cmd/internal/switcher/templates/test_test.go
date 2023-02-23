package templates

import (
	"bytes"
	"log"
	"os"
	"testing"
	"text/template"

	"github.com/metal-stack/metal-core/cmd/internal/switcher/types"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func testTemplate(t *testing.T, tpl *template.Template, c *types.Conf, expectedFilename string) {
	actual := renderToString(t, tpl, c)
	expected := readExpected(t, expectedFilename)
	require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
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
	require.Nil(t, err, "unexpected error when reading testing input")

	err = yaml.Unmarshal(b, &c)
	require.Nil(t, err, "unexpected error when unmarshalling testing input")
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
