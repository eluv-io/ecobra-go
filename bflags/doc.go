/*
	Package bflags implements binding for command flags or command line
	arguments to an object.
	Binding happens through tags annotations on struct types using the `cmd` tag

	Syntax

	Tag are specified using 'cmd' followed by either 'flag' or 'arg':
		flag, name, usage, shorthand, persistent, required, hidden
		arg,  name, usage, order

	sample:
		flag  `cmd:"flag,id,content id,i,true,true,false"`
		arg   `cmd:"arg,id,content id,0"`

	Assuming a string value, the above flag tag is read as (using cobra):
		cmd.PersistentFlags().StringP("id", "i", "", "content id")
		cmd.MarkPersistentFlagRequired("id")
	In case the flag is not persistent and not required, it reads as:
		cmd.Flags().StringP("id", "i", "", "content id")
	Note that flag names are case-sensitive

	The default value is the value of the tagged field. In the example below the
	field Ip of myInput is initialized with net.IPv4(127, 0, 0, 1) before being
 	bound. This makes the flag --ip having a default value of 127.0.0.1.

	For `arg` tags, the order (starting at 0) must be specified for all or none
	of the fields in the struct.

	The library also supports specifying a 'meta' tag, followed by a comma separated
	list of values that are attached as a slice of strings to the 'Annotations' field
	of the resulting FlagBond:
		`meta:"value1,value2"`

	Even though not a frequent usage, the bound flags can be retrieved after binding:
		var c *cobra.Command
		_ = Bind(c, &MyStruct{})
		flags, _ := GetCmdFlagSet(c)
		args, _ := GetCmdArgSet(c)


	Example

		type SimpleTaggedStruct struct {
			Stringval string `cmd:"flag"`
			Ids       id.ID  `cmd:"flag,ids,content ids,i,true,true"`
			Id        id.ID  `cmd:"arg,id,content id,0"`
			QId       id.ID
			Ignore    string
		}

	Complete Usage Sample

		type myInput struct {
			Ip   net.IP `cmd:"flag,ip,node ip,q"`
			Path string `cmd:"arg"`
		}

		func InitFlagsSample() (*cobra.Command, error) {

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
			cmd, err := InitFlagsSample()
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

	Binding to specific 'custom' types

		Binding to specific types is supported through the Flagger interface.
		With an instance fl of Flagger, call bflags.BindCustom(cmd, fl, v)
		See `TestCustomFlag` for a sample implementation.

	Struct implementing flag.Value

		**Pointer** values to those structs can be used as flags.

		Example:

        // workersConfig implements pflag.Value
        type workersConfig struct {
        	QueueSize int
        	Workers   int
        }

        func (u *workersConfig) String() string {
        	bb, err := json.Marshal(u)
        	if err != nil {
        		return err.Error()
        	}
        	return string(bb)
        }

        func (u *workersConfig) Set(s string) error {
        	return json.Unmarshal([]byte(s), u)
        }

        func (u *workersConfig) Type() string {
        	return "workers"
        }

        type TestFlagValueStruct struct {
			// **pointer** !
        	Workers *workersConfig `cmd:"flag"`
        }

        func TestEncodeFlagValue(t *testing.T) {
        	c := &cobra.Command{
        		Use: "dontUse",
        	}
        	sts := &TestFlagValueStruct{
        		Workers: &workersConfig{},
        	}
        	err := Bind(c, sts)
        	require.NoError(t, err)

        	pf := assertFlag(t, c, "Workers")
        	require.Equal(t, &workersConfig{}, sts.Workers)
        	err = pf.Value.Set(`{"QueueSize":50,"Workers":5}`)
        	require.NoError(t, err)
        	wc := &workersConfig{
        		QueueSize: 50,
        		Workers:   5,
        	}

        	require.Equal(t, wc, sts.Workers)
        }

	NOTES
		* inner structs - even anonymous - can be used for bindings BUT the
		  inner struct needs to be initialized otherwise an error is raised
		  since no binding would occur for its fields.


*/
package bflags
