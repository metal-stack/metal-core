package switcher

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func testApplier(t *testing.T, a Applier, expectedFilename string) {
	actual := renderToString(t, a)
	expected := readExpected(t, expectedFilename)
	require.Equal(t, expected, actual, "Wanted: %s, Got: %s", expected, actual)
}

func renderToString(t *testing.T, a Applier) string {
	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	err := a.Render(w)
	require.NoError(t, err, "Couldn't render configuration")
	err = w.Flush()
	require.NoError(t, err, "Couldn't flush writer")
	return b.String()
}

func readConf(t *testing.T, i string) Conf {
	c := Conf{}
	b, err := ioutil.ReadFile(i)
	require.Nil(t, err, "unexpected error when reading testing input")

	err = yaml.Unmarshal(b, &c)
	require.Nil(t, err, "unexpected error when unmarshalling testing input")
	return c
}

func readExpected(t *testing.T, e string) string {
	ex, err := ioutil.ReadFile(e)
	require.NoError(t, err, "Couldn't read %s", e)
	return string(ex)
}

func listTestCases() []string {
	files, err := ioutil.ReadDir("test_data")
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
