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

type rootInput struct {
	Config string `cmd:"flag,config,path to the config file,,true" viper:"config"`
}

type myInput struct {
	Ip   net.IP `cmd:"flag,ip,node ip,q" viper:"node.ip" json:"ip"`
	Path string `cmd:"arg,path,path to file, 0" viper:"path" json:"path"`
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

	b := bflags.NewBinderViper(
		&rootInput{
			Config: "config.toml",
		},
		&cobra.Command{
			Use: "root",
		},
		nil,
		nil,
		bflags.NewViperOpts(vip, false, "config"))
	b.AddCommand(
		bflags.ChildBinder(b, bflags.NewChild(
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
		)))
	if b.Error != nil {
		return b.Error
	}
	err := b.Command.Execute()
	return err
}

// ecobra-go $ go run bflags/examples/viper/viper_sample.go sample /my_path
// sample - input {"ip":"192.168.0.1","path":"/my_path"}
// ecobra-go $ go run bflags/examples/viper/viper_sample.go sample --ip 127.0.0.2 /my_path
// sample - input {"ip":"127.0.0.2","path":"/my_path"}
func main() {
	if err := Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
