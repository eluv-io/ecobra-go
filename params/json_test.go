package params

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func testDir(t *testing.T, prefix string) (path string, cleanup func()) {
	var err error
	path, err = ioutil.TempDir(os.TempDir(), prefix)
	require.NoError(t, err)
	// log.Info("test dir", "path", path)
	cleanup = func() {
		_ = os.RemoveAll(path)
	}
	return path, cleanup
}

func TestJson(t *testing.T) {
	dir, cleanup := testDir(t, "test_json")
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

	js := Json(s)
	mj, err := js.Map()
	require.NoError(t, err)
	require.Equal(t, 2, len(mj))
	require.Equal(t, "blue", mj["color"])
	require.True(t, mj["rgb"].(bool))

	jsp := Json("@" + path)
	mjp, err := jsp.Map()
	require.NoError(t, err)
	require.Equal(t, 2, len(mjp))
	require.Equal(t, "blue", mjp["color"])
	require.True(t, mjp["rgb"].(bool))
}

func TestJson_Interface(t *testing.T) {
	tests := []struct {
		s    string
		want interface{}
	}{
		{`5.4`, 5.4},
		{`true`, true},
		{`"a string"`, "a string"},
		{`{"k1":"v1","k2":"v2"}`, map[string]interface{}{"k1": "v1", "k2": "v2"}},
		{`["v1","v2","v3"]`, []interface{}{"v1", "v2", "v3"}},
	}

	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			val, err := Json(test.s).Interface()
			require.NoError(t, err)
			require.Equal(t, test.want, val)
		})
	}
}

func TestEmptyJson(t *testing.T) {
	js := Json("")
	mj, err := js.Map()
	require.NoError(t, err)
	require.Nil(t, mj)
}

type MyTest struct {
	Size int    `json:"size"`
	Name string `json:"name"`
}

func TestUnmarshalJson(t *testing.T) {
	js := Json(`{"size":10,"name":"john"}`)
	exp := &MyTest{Size: 10, Name: "john"}
	mt := &MyTest{}
	err := js.Unmarshal(mt)
	require.NoError(t, err)
	require.Equal(t, exp, mt)
}
