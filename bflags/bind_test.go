package bflags

import (
	"fmt"
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

func (o *testOpts) run() error {
	o.done = true
	return nil
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
			return opts.run()
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

func TestBinder(t *testing.T) {
	root := NewBinderC(
		&cobra.Command{
			Use:   "test ",
			Short: "root command",
		}).
		AddCommand(
			NewBinder(
				&testOpts{},
				&cobra.Command{
					Use:     "a <domains>",
					Args:    cobra.MinimumNArgs(1),
					Example: "test a x",
				},
				func(opts *testOpts) error {
					return opts.run()
				},
				nil),
			NewBinder(
				&testOpts{},
				&cobra.Command{
					Use:     "b <domains>",
					Args:    cobra.MinimumNArgs(1),
					Example: "test b y",
				},
				func(opts *testOpts) error {
					return opts.run()
				},
				nil))
	require.NoError(t, root.Error)
	require.Equal(t, 2, len(root.Command.Commands()))
}

func TestBinderError(t *testing.T) {
	root := NewBinderC(
		&cobra.Command{
			Use:   "test <domains>",
			Short: "explanation short",
		}).
		AddCommand(
			NewBinderC(
				&cobra.Command{
					Use:   "sub",
					Short: "sub commands",
				}).
				AddCommand(
					NewBinder(
						nil,
						&cobra.Command{
							Use:     "a <domains>",
							Short:   "explanation short",
							Args:    cobra.MinimumNArgs(1),
							Example: "test a b",
						},
						func(opts *testOpts) error {
							return opts.run()
						},
						nil),
					NewBinder(
						nil,
						&cobra.Command{
							Use:     "b <domains>",
							Short:   "explanation short",
							Args:    cobra.MinimumNArgs(1),
							Example: "test a b",
						},
						func(opts *testOpts) error {
							return opts.run()
						},
						nil)))
	require.Error(t, root.Error)
	fmt.Println(root.Error)
}
