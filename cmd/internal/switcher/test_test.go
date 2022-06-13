package switcher

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func testRenderer(t *testing.T, r Renderer, c *Conf, expectedFilename string) {
	actual := renderToString(t, r, c)
	expected := readExpected(t, expectedFilename)
	require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
}

func renderToString(t *testing.T, r Renderer, c *Conf) string {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := r.Render(w, c)
	require.NoError(t, err, "Couldn't render configuration")
	err = w.Flush()
	require.NoError(t, err, "Couldn't flush writer")
	return b.String()
}

func readConf(t *testing.T, i string) Conf {
	c := Conf{}
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
