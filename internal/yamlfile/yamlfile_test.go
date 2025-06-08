package yamlfile

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider(t *testing.T) {
	config := &struct{ Text string }{Text: "foobar"}
	p := Provider(config, "my-path.yaml", "mytag")
	assert.Equal(t, "mytag", p.Tag)
	assert.Same(t, config, p.Defaults)
	assert.NotNil(t, p.File)
}

func TestReadBytes_Errors(t *testing.T) {
	var p YAML[int]
	_, err := p.ReadBytes()
	assert.EqualError(t, err, "yamlfile provider does not support this method")
}

func TestRead_FileNotFound(t *testing.T) {
	var cfg struct{}
	p := Provider(cfg, "non-existing-file.yaml", "mytag")
	_, err := p.Read()
	assert.ErrorContains(t, err, "no such file or directory")
}

func TestRead_YAMLError(t *testing.T) {
	var cfg struct{}
	p := Provider(cfg, writeTempFile(t, "foo: bar:\n"), "mytag")
	_, err := p.Read()
	assert.ErrorContains(t, err, "yaml: mapping values are not allowed in this context")
}

func TestRead_Success(t *testing.T) {
	cfg := struct {
		A string `yaml:"yamlA" mytag:"mytagA"`
		B string `yaml:"yamlB" mytag:"mytagB"`
	}{
		A: "default a",
		B: "default b",
	}

	p := Provider(cfg, writeTempFile(t, "yamlA: yaml a\n"), "mytag")
	got, err := p.Read()
	require.NoError(t, err)

	want := map[string]any{
		"mytagA": "yaml a",
		"mytagB": "default b",
	}
	assert.Equal(t, want, got)
}

func writeTempFile(t *testing.T, content string) string {
	tmpFile, err := os.CreateTemp("", "yamlfile-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, tmpFile.Close())
		assert.NoError(t, os.Remove(tmpFile.Name()))
	})
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	return tmpFile.Name()
}
