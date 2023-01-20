package bflags

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/eluv-io/errors-go"
)

type viperIn1 struct {
	Password  string   `cmd:"flag,password,password for the user's key,x"`
	Name      string   `cmd:"flag,name,name of network" viper:"no.name"`
	NetCert   bool     `cmd:"flag,net-cert,don't add certificate" viper:"net.cert"`
	Networkid int64    `cmd:"flag,networkid,id of the network" viper:"net.networkid"`
	Count     int      `cmd:"flag,count,count of nodes" viper:"net.count"`
	Opts      []string `cmd:"flag,opts,options" viper:"options"`
	Domains   []string `cmd:"arg,domains,name of the ssh domains,0" viper:"domains"`
	done      bool
}

func newViperIn1() *viperIn1 {
	return &viperIn1{
		Password:  "",
		NetCert:   false,
		Name:      "john",
		Networkid: 2,
		Count:     3,
		Domains:   nil,
	}
}

func TestPreloadedViperBindingFlags(t *testing.T) {
	toml := `
domains = "a,b"
[net]
  cert = true           # use cert
  networkid = 955302    # Network identifier
`
	//vip.AutomaticEnv()

	inOpts := newViperIn1()
	c := &cobra.Command{
		Use: "dontUse",
	}
	var err error

	// The usual order seems:
	// - bind flags
	// - then read config
	// which does not work for us since viper does not apply the updated value to flags
	// see also:
	//    Same flag name for different cobra commands does not work #233
	//    https://github.com/spf13/viper/issues/233
	// And what seems the normal usage to get a value is then to always use viper
	// func GetXXX which (see Viper.find):
	// - first search in its maps
	// - then in cobra flags
	// - then in env

	// In our case we're only interested in flags, so:
	// - instead read the config first
	// - then bind flags and set the viper's value during binding
	vip := viper.New()
	vip.SetConfigType("toml")
	//vip.SetConfigName()
	//vip.AddConfigPath()
	err = vip.ReadConfig(bytes.NewReader([]byte(toml)))
	require.NoError(t, err)

	require.Equal(t, int64(955302), vip.GetInt64("net.networkid"))

	err = BindCustomViper(c, nil, inOpts, NewViperOpts(vip, true, ""))
	require.NoError(t, err)

	pfn := assertFlag(t, c, "networkid")
	require.Equal(t, "john", inOpts.Name)
	require.Equal(t, 3, inOpts.Count)
	require.Equal(t, int64(955302), inOpts.Networkid)
	require.True(t, inOpts.NetCert)
	require.Equal(t, []string{"a", "b"}, inOpts.Domains)

	err = pfn.Value.Set("956000")
	require.NoError(t, err)
	require.Equal(t, int64(956000), inOpts.Networkid)

	fmt.Println("all viper keys:", strings.Join(vip.AllKeys(), ","))
}

func TestPreloadedViperBinding(t *testing.T) {
	type testCase struct {
		toml     string
		args     []string
		expected *viperIn1
	}

	for i, tc := range []*testCase{
		{
			toml: `
              domains = "a,b"
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "--networkid", "956000", "x", "y"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 956000,
				Count:     4,
				Domains:   []string{"x", "y"},
				Opts:      nil,
				done:      true,
			},
		},
		{
			toml: `
              domains = "a,b"
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "--networkid", "956000"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 956000,
				Count:     4,
				Domains:   []string{"a", "b"},
				Opts:      nil,
				done:      true,
			},
		},
		{
			toml: `
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "--networkid", "956000", "x", "y"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 956000,
				Count:     4,
				Domains:   []string{"x", "y"},
				Opts:      nil,
				done:      true,
			},
		},
		{
			toml: `
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "x", "y"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 955302,
				Count:     4,
				Domains:   []string{"x", "y"},
				Opts:      nil,
				done:      true,
			},
		},
		{
			toml: `
              options = "a,b"
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "x", "y", "--opts", "x,y"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 955302,
				Count:     4,
				Domains:   []string{"x", "y"},
				Opts:      []string{"x", "y"},
				done:      true,
			},
		},
		{
			toml: `
              options = "a,b"
              [net]
                cert = true           # use cert
                networkid = 955302    # Network identifier
              `,
			args: []string{"xx", "x", "y"},
			expected: &viperIn1{
				Password:  "",
				Name:      "john",
				NetCert:   true,
				Networkid: 955302,
				Count:     4,
				Domains:   []string{"x", "y"},
				Opts:      []string{"a", "b"},
				done:      true,
			},
		},
	} {
		vip := viper.New()
		vip.SetConfigType("toml")
		err := vip.ReadConfig(bytes.NewReader([]byte(tc.toml)))
		require.NoError(t, err, "case %d", i)

		in := newViperIn1()
		cmd, err := BindRunEViper(
			in,
			&cobra.Command{
				Use:  "test <domains>",
				Args: cobra.MinimumNArgs(0),
			},
			func(opts *viperIn1) error {
				opts.Count += 1
				opts.done = true

				j, err := json.Marshal(opts)
				require.NoError(t, err)
				fmt.Println("in", string(j))

				return nil
			},
			nil,
			NewViperOpts(vip, true, ""))
		require.NoError(t, err, "case %d", i)
		require.NotNil(t, cmd)

		os.Args = tc.args
		err = cmd.Execute()
		require.NoError(t, err, "case %d", i)

		require.Equal(t, tc.expected, in, "case %d", i)
		require.True(t, in.done, "case %d", i)
	}
}

type viperIn2 struct {
	Config    string   `cmd:"flag,config,path to the config file,,true" viper:"config"`
	Password  string   `cmd:"flag,password,password for the user's key,x"`
	Name      string   `cmd:"flag,name,name of network" viper:"no.name"`
	NetCert   bool     `cmd:"flag,net-cert,don't add certificate" viper:"net.cert"`
	Networkid int64    `cmd:"flag,networkid,id of the network" viper:"net.networkid"`
	Count     int      `cmd:"flag,count,count of nodes" viper:"net.count"`
	Opts      []string `cmd:"flag,opts,options" viper:"options"`
	Domains   []string `cmd:"arg,domains,name of the ssh domains,0" viper:"domains"`
	done      bool
}

func newViperIn2() *viperIn2 {
	return &viperIn2{
		Config:    "config.toml",
		Password:  "",
		NetCert:   false,
		Name:      "john",
		Networkid: 2,
		Count:     3,
		Domains:   nil,
	}
}

func TestViperBindingFlagsConfig(t *testing.T) {
	defaultToml := `
domains = "a,b"
[net]
  cert = true           # use cert
  networkid = 955302    # Network identifier
`
	newToml := `
domains = "a,b,c"
[net]
  cert = true           # use cert
  networkid = 955333    # Network identifier
`
	tmp, err := os.CreateTemp("", "test-viper-binding-flags")
	require.NoError(t, err)
	folder := tmp.Name()
	_ = os.Remove(tmp.Name())
	err = os.MkdirAll(folder, os.ModePerm)
	require.NoError(t, err)
	defer errors.Ignore(func() error { return os.RemoveAll(folder) })

	folder1 := filepath.Join(folder, "default")
	err = os.MkdirAll(folder1, os.ModePerm)
	require.NoError(t, err)
	defaultTomlFile := filepath.Join(folder1, "config.toml")
	err = os.WriteFile(defaultTomlFile, []byte(defaultToml), os.ModePerm)
	require.NoError(t, err)

	folder2 := filepath.Join(folder, "other")
	err = os.MkdirAll(folder2, os.ModePerm)
	require.NoError(t, err)
	newTomlFile := filepath.Join(folder2, "config.toml")
	err = os.WriteFile(newTomlFile, []byte(newToml), os.ModePerm)
	require.NoError(t, err)

	type testCase struct {
		descr    string
		config   string
		args     []string
		expected *viperIn2
	}
	for _, tc := range []*testCase{
		{descr: "use default config", args: []string{},
			config: defaultTomlFile,
			expected: &viperIn2{
				Config:    defaultTomlFile,
				Password:  "",
				NetCert:   true,
				Name:      "john",
				Networkid: 955302,
				Count:     4,
				Opts:      []string{},
				Domains:   []string{"a", "b"},
				done:      true,
			}},
		{descr: "config from flag", args: []string{"--config", newTomlFile},
			config: defaultTomlFile,
			expected: &viperIn2{
				Config:    newTomlFile,
				Password:  "",
				NetCert:   true,
				Name:      "john",
				Networkid: 955333,
				Count:     4,
				Opts:      []string{},
				Domains:   []string{"a", "b", "c"},
				done:      true,
			}},
	} {
		inOpts := newViperIn2()
		if tc.config != "" {
			inOpts.Config = tc.config
		}

		c := &cobra.Command{
			Use: "dontUse",
		}

		vip := viper.New()
		vip.SetConfigType("toml")

		require.Equal(t, int64(0), vip.GetInt64("net.networkid"))

		c, err = BindRunEViper(
			inOpts,
			c,
			func(in *viperIn2) error {
				in.done = true
				in.Count++
				return nil
			},
			nil,
			NewViperOpts(vip, false, "config"))
		require.NoError(t, err, "%s", tc.descr)
		// flags were bound: we have default values
		{
			require.Equal(t, defaultTomlFile, inOpts.Config)
			require.Equal(t, "john", inOpts.Name)
			require.Equal(t, 3, inOpts.Count)
			require.Equal(t, int64(2), inOpts.Networkid)
			require.False(t, inOpts.NetCert)
			require.Equal(t, 0, len(inOpts.Domains))
		}

		os.Args = append([]string{"dummy"}, tc.args...)
		err = c.Execute()
		require.NoError(t, err, "%s", tc.descr)
		require.Equal(t, tc.expected, inOpts, "%s", tc.descr)
	}
}

type rootIn struct {
	Config   string `cmd:"flag,config,path to the config file,,true" viper:"config"`
	Password string `cmd:"flag,password,password for the user's key,x"`
}

type netSetIn struct {
	Ip   net.IP `cmd:"flag,ip,node ip" viper:"node.ip" json:"ip"`
	Name string `cmd:"flag,name,node name" viper:"node.name" json:"name"`
	Path string `cmd:"arg,path,path to file, 0" viper:"path" json:"path"`
}

func TestViperBindingChildConfig(t *testing.T) {

	tmp, err := os.CreateTemp("", "test-viper-binding-flags")
	require.NoError(t, err)
	folder := tmp.Name()
	_ = os.Remove(tmp.Name())
	err = os.MkdirAll(folder, os.ModePerm)
	require.NoError(t, err)
	defer errors.Ignore(func() error { return os.RemoveAll(folder) })

	folder1 := filepath.Join(folder, "default")
	err = os.MkdirAll(folder1, os.ModePerm)
	require.NoError(t, err)
	defaultTomlFile := filepath.Join(folder1, "config.toml")

	defaultToml := `
[node]
  name = "local-net"
  ip = "192.168.1.1"
`
	err = os.WriteFile(defaultTomlFile, []byte(defaultToml), os.ModePerm)
	require.NoError(t, err)

	folder2 := filepath.Join(folder, "other")
	err = os.MkdirAll(folder2, os.ModePerm)
	require.NoError(t, err)
	newTomlFile := filepath.Join(folder2, "config.toml")
	newToml := `
[node]
  #name = "local-net"
  ip = "10.1.1.1"
`
	err = os.WriteFile(newTomlFile, []byte(newToml), os.ModePerm)
	require.NoError(t, err)

	var out *netSetIn
	binder := func() *Binder {
		vip := viper.New()
		vip.SetConfigType("toml")

		b := NewBinderViper(
			&rootIn{
				Config: defaultTomlFile,
			},
			&cobra.Command{
				Use: "root",
			},
			nil,
			nil,
			NewViperOpts(vip, false, "config"))
		b.AddCommand(
			ChildBinder(b, &Child[any]{
				C: &cobra.Command{
					Use: "file",
				},
			}),
			ChildBinder(b, NewChild[any](
				nil,
				&cobra.Command{
					Use: "net",
				},
				nil)).AddCommand(
				ChildBinder(b, NewChild(
					&netSetIn{
						Ip:   net.IPv4(127, 0, 0, 1),
						Name: "local",
						Path: "/tmp1",
					},
					&cobra.Command{
						Use:  "set",
						Args: cobra.MinimumNArgs(1),
					},
					func(in *netSetIn) error {
						bb, err := json.MarshalIndent(in, "", "  ")
						if err != nil {
							return err
						}
						fmt.Println(string(bb))
						vout := *in
						out = &vout
						return nil
					},
				)),
				// also possible but require explicit type declaration (as [netSetIn])
				//ChildBinder(b, &Child[netSetIn]{
				//	In: &netSetIn{
				//		Ip:   net.IPv4(127, 0, 0, 1),
				//		Path: "/tmp1",
				//	},
				//	C: &cobra.Command{
				//		Use:  "set",
				//		Args: cobra.MinimumNArgs(1),
				//	},
				//	RunE: func(in *netSetIn) error {
				//		return nil
				//	},
				//}),
			))
		require.NoError(t, b.Error)
		return b
	}
	b := binder()
	os.Args = []string{"dummy", "net", "set", "/my-path"}
	err = b.Command.Execute()
	require.NoError(t, err)
	require.Equal(t, &netSetIn{
		Ip:   net.IPv4(192, 168, 1, 1),
		Name: "local-net",
		Path: "/my-path",
	}, out)

	b = binder()
	os.Args = []string{"dummy", "net", "set", "/my-other-path", "--config", newTomlFile}
	err = b.Command.Execute()
	require.NoError(t, err)
	require.Equal(t, &netSetIn{
		Ip:   net.IPv4(10, 1, 1, 1),
		Name: "local",
		Path: "/my-other-path",
	}, out)

}
