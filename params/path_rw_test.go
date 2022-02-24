package params

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/eluv-io/ecobra-go/bflags"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func assertFlag(t *testing.T, c *cobra.Command, name string) *pflag.Flag {
	flags, err := bflags.GetCmdFlagSet(c)
	require.NoError(t, err)
	_, ok := flags.Get(name)
	require.True(t, ok)
	pf := c.Flag(name)
	require.NotNil(t, pf)
	return pf
}

type TestReader struct {
	Name string        `cmd:"flag"`
	Path *PathOrReader `cmd:"flag"`
}

func TestPathOrReader(t *testing.T) {
	// PathOrReader not initialized fails
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestReader{}
	err := bflags.Bind(c, sts)
	require.NoError(t, err)
	pfr := assertFlag(t, c, "Path")
	require.Nil(t, sts.Path)
	err = pfr.Value.Set("bla")
	require.Error(t, err) // Path must be initialized

	// PathOrReader initialized - test with path
	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestReader{
		Path: &PathOrReader{},
	}
	err = bflags.Bind(c, sts)
	require.NoError(t, err)

	// verify a normal flag works
	pfName := assertFlag(t, c, "Name")
	require.Empty(t, sts.Name)
	err = pfName.Value.Set("john")
	require.NoError(t, err)
	require.Equal(t, "john", sts.Name)

	// now the reader
	dir, cleanup := testDir(t, "test_reader")
	defer cleanup()

	s := "hello world"
	path := filepath.Join(dir, "f.txt")
	err = ioutil.WriteFile(path, []byte(s), 0700)
	require.NoError(t, err)

	pfRead := assertFlag(t, c, "Path")
	require.Nil(t, sts.Path.Read)
	err = pfRead.Value.Set(path)
	require.NoError(t, err)
	require.NotEmpty(t, sts.Path.Path)
	require.Nil(t, sts.Path.Read)
	rd, err := sts.Path.Open()
	require.NoError(t, err)
	require.NotNil(t, sts.Path.Read)

	bb, err := ioutil.ReadAll(rd)
	_ = rd.Close()
	require.NoError(t, err)
	require.Equal(t, s, string(bb))

	// PathOrReader initialized - test with reader
	c = &cobra.Command{
		Use: "dontUse",
	}
	frd, err := os.Open(path)
	require.NoError(t, err)
	defer func() { _ = frd.Close() }()

	sts = &TestReader{
		Path: &PathOrReader{
			Read: frd,
		},
	}
	err = bflags.Bind(c, sts)
	require.NoError(t, err)
	pfRead = assertFlag(t, c, "Path")
	require.NotNil(t, sts.Path.Read)
	require.Empty(t, sts.Path.Path)
	err = pfRead.Value.Set(path)
	require.Error(t, err)
	rd2, err := sts.Path.Open()
	require.NoError(t, err)
	bb, err = ioutil.ReadAll(rd2)
	require.NoError(t, err)
	require.Equal(t, s, string(bb))
}

type TestWriter struct {
	Name string        `cmd:"flag"`
	Path *PathOrWriter `cmd:"flag"`
}

func TestPathOrWriter(t *testing.T) {
	// PathOrWriter not initialized fails
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &TestWriter{}
	err := bflags.Bind(c, sts)
	require.NoError(t, err)
	pfr := assertFlag(t, c, "Path")
	require.Nil(t, sts.Path)
	err = pfr.Value.Set("bla")
	require.Error(t, err) // Path must be initialized

	// PathOrWriter initialized - test with path
	c = &cobra.Command{
		Use: "dontUse",
	}
	sts = &TestWriter{
		Path: &PathOrWriter{},
	}
	err = bflags.Bind(c, sts)
	require.NoError(t, err)

	// verify a normal flag works
	pfName := assertFlag(t, c, "Name")
	require.Empty(t, sts.Name)
	err = pfName.Value.Set("john")
	require.NoError(t, err)
	require.Equal(t, "john", sts.Name)

	// now the writer
	dir, cleanup := testDir(t, "test_writer")
	defer cleanup()
	s := "hello world"
	path := filepath.Join(dir, "f.txt")

	pfWrite := assertFlag(t, c, "Path")
	require.Nil(t, sts.Path.Write)
	err = pfWrite.Value.Set(path)
	require.NoError(t, err)
	require.NotEmpty(t, sts.Path.Path)
	require.Nil(t, sts.Path.Write)
	wr, err := sts.Path.Create()
	require.NoError(t, err)
	require.NotNil(t, sts.Path.Write)

	n, err := io.Copy(wr, bytes.NewBuffer([]byte(s)))
	_ = wr.Close()
	require.NoError(t, err)
	require.Equal(t, len(s), int(n))
	bb, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, s, string(bb))

	// PathOrWriter initialized - test with writer
	c = &cobra.Command{
		Use: "dontUse",
	}
	path = filepath.Join(dir, "f2.txt")
	wrd, err := os.Create(path)
	require.NoError(t, err)
	defer func() { _ = wrd.Close() }()

	sts = &TestWriter{
		Path: &PathOrWriter{
			Write: wrd,
		},
	}
	err = bflags.Bind(c, sts)
	require.NoError(t, err)
	pfWrite = assertFlag(t, c, "Path")
	require.NotNil(t, sts.Path.Write)
	require.Empty(t, sts.Path.Path)
	err = pfWrite.Value.Set(path)
	require.Error(t, err)
	wrd2, err := sts.Path.Create()
	require.NoError(t, err)

	n, err = io.Copy(wrd2, bytes.NewBuffer([]byte(s)))
	require.NoError(t, err)
	require.Equal(t, len(s), int(n))
	bb, err = ioutil.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, s, string(bb))
}
