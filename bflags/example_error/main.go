// Package main provides an example of using bflags.BindRunE and how to silence
// errors or remove stack traces in error returned from the runE function.
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/bflags"
	"github.com/eluv-io/errors-go"
)

type myInput struct {
	Ip           net.IP `cmd:"flag,ip,node ip,q"`
	Silent       bool   `cmd:"flag,silent,silence errors"`
	NoStackTrace bool   `cmd:"flag,no-trace,remove stack-trace in reported errors"`
	Path         string `cmd:"arg"`
}

// SilenceErrors is the interface for silencing errors in cobra.Command
func (i *myInput) SilenceErrors() bool {
	return i.Silent
}

// NoTrace is the interface for telling bindRunE to clear stack traces in returned error
func (i *myInput) NoTrace() bool {
	return i.NoStackTrace
}

func (i *myInput) run() error {
	e := errors.Template("run", errors.K.Invalid, "reason", "always fails")
	bb, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if !i.Silent {
		fmt.Printf("sample - input %s\n", string(bb))
	}

	return e()
}

func Execute() error {
	cmd, err := bflags.BindRunE(
		&myInput{
			Ip:   net.IPv4(127, 0, 0, 1),
			Path: "/tmp",
		},
		&cobra.Command{
			Use:   "sample /path",
			Short: "run sample",
		},
		func(in *myInput) error { return in.run() },
		nil)
	if err != nil {
		return err
	}
	err = cmd.Execute()
	return err
}

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
