// Package main provides an example in how to use bflags.BindRunE
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/bflags"
)

type myInput struct {
	Ip   net.IP `cmd:"flag,ip,node ip,q"`
	Path string `cmd:"arg"`
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
		func(in *myInput) error {
			bb, err := json.Marshal(in)
			if err != nil {
				return err
			}
			fmt.Printf("sample - input %s\n", string(bb))
			return nil
		},
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
