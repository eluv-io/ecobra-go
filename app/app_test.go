package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePositional(t *testing.T) {
	type test struct {
		val    string
		ename  string
		eint0  int
		eint1  int
		hasErr bool
	}
	tests := []*test{
		{"NoArgs       ", "NoArgs", 0, 0, false},
		{"OnlyValidArgs", "OnlyValidArgs", 0, 0, false},
		{"ExactArgs(1)", "ExactArgs", 1, 0, false},
		{"RangeArgs(1,2)", "RangeArgs", 1, 2, false},
		{"ExactArgs(1", "", 0, 0, true},
	}

	for _, te := range tests {
		v, i0, i1, err := parsePositional(te.val)
		require.Equal(t, te.hasErr, err != nil)
		require.Equal(t, te.ename, v)
		require.Equal(t, te.eint0, i0)
		require.Equal(t, te.eint1, i1)
	}

}
