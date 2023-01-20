package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/eluv-io/ecobra-go/bflags"
)

type myInput struct {
	Ip   net.IP `cmd:"flag,ip,node ip,q" viper:"node.ip"`
	Path string `cmd:"arg,path,path to file, 0" viper:"path"`
}

func Execute() error {

	vip := viper.New()
	vip.SetConfigType("toml")
	vip.SetConfigName("config")

	pwd, _ := os.Getwd()
	vip.AddConfigPath(".")
	vip.AddConfigPath(pwd)
	// assuming we run from source top dir
	vip.AddConfigPath(filepath.Join(pwd, "bflags/examples/viper"))

	// in this example, viper is loaded before bindings
	err := vip.ReadInConfig()
	if err != nil {
		return err
	}

	cmd, err := bflags.BindRunEViper(
		&myInput{
			Ip:   net.IPv4(127, 0, 0, 1),
			Path: "/tmp1",
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
		nil,
		&bflags.ViperOpts{Viper: vip, Loaded: true})
	if err != nil {
		return err
	}
	err = cmd.Execute()
	return err
}

// ecobra-go $ go run bflags/examples/viper_preload/viper_sample.go --ip 127.0.0.2 /my_path
// sample - input {"Ip":"127.0.0.2","Path":"/my_path"}
func main() {
	if err := Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
