package bflags

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type testOpts struct {
	Password string   `cmd:"flag,password,password for the user's key,x"`
	NoCert   bool     `cmd:"flag,no-cert,don't add certificate to the ssh-agent"`
	Domains  []string `cmd:"arg,domains,name of the ssh domains,0"`
	done     bool
}

func TestBind(t *testing.T) {
	in := &testOpts{}
	cmd, err := BindRunE(
		in,
		&cobra.Command{
			Use:     "test <domains>",
			Short:   "explanation short",
			Args:    cobra.MinimumNArgs(1),
			Example: "test a b",
		},
		func(opts *testOpts) error {
			opts.done = true
			return nil
		},
		nil)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	_ = cmd.Help()
	os.Args = []string{"xx", "--no-cert", "x", "y"}
	err = cmd.Execute()
	require.NoError(t, err)
	require.True(t, in.NoCert)
	require.True(t, in.done)
	require.Equal(t, []string{"x", "y"}, in.Domains)
}
