package bflags

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
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

func TestBindNil(t *testing.T) {
	in := &testOpts{}
	_, err := BindRunE(
		in,
		nil,
		func(opts *testOpts) error {
			return opts.run()
		},
		nil)
	require.Error(t, err)
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

type silenceOpts struct {
	*testOpts
}

func (o *silenceOpts) run() error {
	o.done = true
	return io.EOF
}

func (o *silenceOpts) SilenceErrors() bool {
	return true
}

func TestBindSilenceErrors(t *testing.T) {
	in := &silenceOpts{
		testOpts: &testOpts{},
	}
	cmd, err := BindRunE(
		in,
		&cobra.Command{
			Use:     "test <domains>",
			Short:   "explanation short",
			Args:    cobra.MinimumNArgs(1),
			Example: "test a b",
		},
		func(opts *silenceOpts) error {
			return opts.run()
		},
		nil)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	out := bytes.NewBuffer([]byte{})
	cmd.SetOut(out)
	cmd.SetErr(out)

	err = cmd.Execute()
	require.Error(t, err)
	require.True(t, in.done)
	require.Empty(t, out.String())
}

type errorsOpts struct {
	*testOpts
}

func (o *errorsOpts) run() error {
	o.done = true
	return errors.E("run", errors.K.Invalid, "reason", "testing")
}

func (o *errorsOpts) NoTrace() bool {
	return true
}

func TestBindErrorsNoTrace(t *testing.T) {
	in := &errorsOpts{
		testOpts: &testOpts{},
	}
	cmd, err := BindRunE(
		in,
		&cobra.Command{
			Use:     "test <domains>",
			Short:   "explanation short",
			Args:    cobra.MinimumNArgs(1),
			Example: "test a b",
		},
		func(opts *errorsOpts) error {
			return opts.run()
		},
		nil)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	out := bytes.NewBuffer([]byte{})
	cmd.SetOut(out)
	cmd.SetErr(out)

	err = cmd.Execute()
	require.Error(t, err)
	require.True(t, in.done)
	require.False(t,
		strings.Contains(out.String(), "github.com/eluv-io/ecobra-go/bflags/bind_test.go"),
		out.String())
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
