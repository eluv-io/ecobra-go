package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/eluv-io/errors-go"
	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/bflags"
)

type myInput struct {
	Ip   net.IP `cmd:"flag,ip,node ip,q"`
	Path string `cmd:"arg"`
}

func initFlagsSample() (*cobra.Command, error) {

	var cmdFlagsSample = &cobra.Command{
		Use:   "sample /path",
		Short: "run sample",
		RunE:  runSample,
	}
	// &myInput instance 'my' here represents the default values of all
	// flags bound to it
	my := &myInput{
		Ip:   net.IPv4(127, 0, 0, 1),
		Path: "/tmp",
	}

	err := bflags.Bind(cmdFlagsSample, my)
	// this is equivalent to
	//   cmdFlagsSample.Flags().IPVarP(&my.Ip, "ip", "q", my.Ip, "node ip")
	// additionally if the flag was required
	//   _ = cmdFlagsSample.MarkFlagRequired("ip")
	// or hidden:
	//   cmdFlagsSample.Flag("ip").Hidden = true

	if err != nil {
		return nil, err
	}
	return cmdFlagsSample, nil
}

func runSample(cmd *cobra.Command, args []string) error {
	// this is required to bind args parameters to the 'arg' fields of the input
	m, err := bflags.SetArgs(cmd, args)
	if err != nil {
		return err
	}
	my, ok := m.(*myInput)
	if !ok {
		return errors.E("runSample", "wrong input", m)
	}
	bb, err := json.Marshal(my)
	if err != nil {
		return err
	}
	fmt.Printf("sample - input %s\n", string(string(bb)))
	return nil
}

func Execute() error {
	cmd, err := initFlagsSample()
	if err != nil {
		return err
	}
	err = cmd.Execute()
	return err
}

func main() {
	if err := Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
