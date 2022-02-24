package bflags

import (
	"flag"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type path []string

type fl struct{}

func (*fl) Bind(t reflect.Type) bool {
	return t == reflect.TypeOf(path(nil)) || t == reflect.TypeOf(&path{})
}

func (*fl) Flag(val interface{}) *Flagged {

	switch val := val.(type) {
	case path:
		p := new(path)
		return &Flagged{
			Ptr:  p,
			Flag: newPathValue(val, p),
		}
	case *path:
		return &Flagged{
			Ptr:  val,
			Flag: newPathValue(*val, val),
		}
	}
	return nil
}

type pathValue path

var _ flag.Value = (*pathValue)(nil)

func newPathValue(val path, p *path) *pathValue {
	*p = val
	return (*pathValue)(p)
}

func (b *pathValue) Set(s string) error {
	pp := strings.Split(s, "/")
	*b = pathValue(pp)
	return nil
}

func (b *pathValue) Type() string {
	return "$pathValue"
}

func (b *pathValue) String() string {
	return strings.Join([]string(*b), "/")
}

type MyObject struct {
	Me   string `cmd:"flag,name"`
	Path path   `cmd:"flag,path"`
}

func TestCustomFlag(t *testing.T) {
	c := &cobra.Command{
		Use: "dontUse",
	}
	sts := &MyObject{}

	err := BindCustom(c, &fl{}, sts)
	require.NoError(t, err)

	pfn := assertFlag(t, c, "name")
	require.Empty(t, sts.Me)
	err = pfn.Value.Set("bob")
	require.NoError(t, err)
	require.Equal(t, "bob", sts.Me)

	pfp := assertFlag(t, c, "path")
	require.Empty(t, sts.Path)
	err = pfp.Value.Set("path/to/bla")
	require.NoError(t, err)
	require.EqualValues(t, []string{"path", "to", "bla"}, sts.Path)

}
