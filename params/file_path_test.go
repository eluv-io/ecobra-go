package params

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPipe(t *testing.T) {
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer func() { _ = r.Close() }()
	os.Stdin = r

	fp := FilePath("-")
	in, err := fp.Open()
	require.NoError(t, err)
	var bb []byte
	_, err = fmt.Fprintf(w, "hello")
	require.NoError(t, err)
	go func() {
		_ = w.Close()
	}()
	bb, err = ioutil.ReadAll(in)

	//fmt.Println(">>", string(bb), "<<", n)
	require.Equal(t, "hello", string(bb))
}

func TestFilePath(t *testing.T) {
	dir, cleanup := testDir(t, "test_fpath")
	defer cleanup()

	mm := make(map[string]interface{})
	mm["color"] = "blue"
	mm["rgb"] = true
	bb, err := json.Marshal(mm)
	require.NoError(t, err)
	s := string(bb)

	path := filepath.Join(dir, "f.json")
	err = ioutil.WriteFile(path, []byte(s), 0700)
	require.NoError(t, err)

	fp := FilePath(path)
	f, err := fp.Open()
	require.NoError(t, err)

	bb, err = ioutil.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, s, string(bb))
}

func TestFilePathSet(t *testing.T) {
	p := FilePath("hi")
	fp := &p
	err := fp.Set("ho")
	require.NoError(t, err)
	require.Equal(t, "ho", fp.String())
}
