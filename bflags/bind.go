package bflags

import (
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/eluv-io/errors-go"
)

// Bind binds the flags retrieved in tags of the given struct v as flags
// or args of the given command.
// c the command
//
// v expected to be a struct.
//
// Tag are specified using 'cmd' followed by either 'flag' or 'arg':
//
//	`cmd:"flag,id,content id, i,true,true,true"`
//	`cmd:"arg,id,content id,0"`
//
// are read as:
//
//	flag: name, usage, shorthand, persistent, required, hidden
//	arg: name, usage, order
//
// Attributes with default value on the right side of the expression can be omitted:
//
//	`cmd:"flag,id,content id, i"`
//
// A 'meta' tag can be added to convey annotations with the bound tag:
//
//	`cmd:"flag,config,file name,c" meta:"file,non-empty"`
func Bind(c *cobra.Command, v interface{}) error {
	return BindCustom(c, nil, v)
}

// BindCustom is like Bind and allows a custom Flagger
func BindCustom(c *cobra.Command, f Flagger, v interface{}) error {
	if v == nil {
		setCmdInput(c, nil)
		return nil
	}
	e := newFlagsBinder(c, f)

	err := e.bind(v, bindOpts{})
	if err != nil {
		path := append([]string{}, c.Name())
		r := c.Parent()
		for r != nil {
			path = append(path, "")
			copy(path[1:], path[0:])
			path[0] = r.Name()
			r = r.Parent()
		}
		return errors.E("bindToStruct", err,
			"command", c.Name(),
			"path", strings.Join(path, "/"))
	}

	e.Reset(nil, nil)
	bindStatePool.Put(e)

	return nil
}

// isSliceValue returns true if the Value of the flag has a type (string) ending
// with 'Slice' which is a convention respected over the pflag package.
func isSliceValue(f *flag.Flag) bool {
	if f == nil {
		return false
	}
	return strings.HasSuffix(f.Value.Type(), "Slice")
}

// SetArgs sets the args to the 'arg' fields of the value previously bound as the
// input of the command. The input must have previously been bound to the command
// like so:
//
//	input := &MyStruct{}
//	err := bflags.BindCustom(cmd, elvflags.Flags, input)
//
// The input is returned if no error occurred.
func SetArgs(c *cobra.Command, args []string) (interface{}, error) {
	ex := errors.Template("setArgs")
	if log.IsDebug() {
		// report found flags and args in debug mode
		cmdflags, err := GetCmdFlagSet(c)
		s := "null"
		if err == nil {
			bb, err2 := json.Marshal(cmdflags)
			if err2 != nil {
				log.Warn("error marshalling cmdFlags", err2)
			}
			s = string(bb)
		}
		log.Debug("set args - bound flags", "flags", s)
		argflags, err := GetCmdArgSet(c)
		s = "null"
		if err == nil {
			bb, err2 := json.Marshal(argflags)
			if err2 != nil {
				log.Warn("error marshalling argFlags", err2)
			}
			s = string(bb)
		}
		log.Debug("set args - bound args", "args", s)
	}

	if len(args) > 0 {
		argset, err := GetCmdArgSet(c)
		if err != nil {
			return nil, err
		}
		for i, arg := range args {
			if i < len(argset.Flags) {
				f := c.Flags().Lookup(string(argset.Flags[i].Name))
				if f == nil {
					return nil, ex(errors.K.NotExist, "name", argset.Flags[i].Name)
				}
				if arg == "" {
					continue
				}
				// support for variadic args with slices arg
				if i == len(argset.Flags)-1 && len(args) > len(argset.Flags) && isSliceValue(f) {
					arg = strings.Join(args[i:], ",")
				}
				err = f.Value.Set(arg)
				if err != nil {
					return nil, ex(err)
				}
			}
		}
	}

	if log.IsDebug() {
		// reconstruct command line from all
		cmds := make([]string, 0, 16)
		cc := c
		for cc != nil {
			cmds = append(cmds, "")
			copy(cmds[1:], cmds[0:])
			cmds[0] = cc.Name()
			cc = cc.Parent()
		}

		cmdflags, err := GetCmdFlagSet(c)
		if err == nil {
			for _, f := range cmdflags {
				cmds = append(cmds, f.CmdString()...)
			}
		}
		argflags, err := GetCmdArgSet(c)
		if err == nil {
			for _, f := range argflags.Flags {
				cmds = append(cmds, f.CmdString()...)
			}
		}
		log.Debug(strings.Join(cmds, " "))
	}

	v, _ := GetCmdInput(c)
	return v, nil
}

// SetupCmdArgs configures and returns the input struct bound to the provided
// command with the given arguments.
// * If the typ parameter is not nil the type of the input is verified
// * if the input has a function 'Validate() error', the function is called
// The input must have previously been bound to the command like so:
//
//	input := &MyStruct{}
//	err := bflags.BindCustom(cmd, elvflags.Flags, input)
//
// The input is returned if no error occurred.
func SetupCmdArgs(cmd *cobra.Command, args []string, typ reflect.Type) (interface{}, error) {
	e := errors.Template("setupCmd", errors.K.Invalid)

	m, err := SetArgs(cmd, args)
	if err != nil {
		return nil, e(err)
	}
	if typ != nil && typ != reflect.TypeOf(m) {
		return nil, e("reason", "wrong input", "input", m)
	}
	type validatable interface {
		Validate() error
	}
	if val, ok := m.(validatable); ok {
		err = val.Validate()
		if err != nil {
			return nil, e(err)
		}
	}
	// no usage from now on
	cmd.SilenceUsage = true
	cmd.Root().SilenceUsage = true

	return m, nil
}

// GetFlagArgSet returns a map[string]string of flags and args that were set for
// the given command. This function has to be called after SetArgs.
func GetFlagArgSet(c *cobra.Command) map[string]string {
	ret := make(map[string]string)

	if cmdflags, err := GetCmdFlagSet(c); err == nil {
		for name, fl := range cmdflags {
			// for flags: cmdString returns ["--flag", "value"] except for bool
			ss := fl.cmdString(true)
			switch len(ss) {
			case 0:
				continue
			case 1:
				// should not happen since we ask for full flag
				ndx := strings.Index(ss[0], "=")
				if ndx > 0 {
					ret[string(name)] = ss[0][ndx+1:]
				}
			default:
				ret[string(name)] = ss[1]
			}
		}
	}
	if argflags, err := GetCmdArgSet(c); err == nil {
		for _, fl := range argflags.Flags {
			ss := fl.CmdString()
			if len(ss) == 0 {
				continue
			}
			ret[string(fl.Name)] = ss[0]
		}
	}
	return ret
}

// BindRunE binds the input parameter to the given command and sets the runE
// parameter function as the function invoked by the RunE function of the cobra command.
func BindRunE[T any](input *T, cmd *cobra.Command, runE func(*T) error, f Flagger) (*cobra.Command, error) {
	e := errors.Template("BindRunE", errors.K.Invalid,
		"input", input,
		"command", cmd.Use)
	if cmd == nil {
		return nil, e("reason", "nil command  not allowed")
	}
	if input == nil && runE != nil {
		return nil, e("reason", "nil input with non nil runE function")
	}
	if input == nil {
		return cmd, nil
	}
	if runE == nil {
		return nil, e("reason", "nil runE function")
	}

	if reflect.TypeOf(input).Kind() != reflect.Ptr ||
		reflect.ValueOf(input).Kind() != reflect.Ptr ||
		reflect.ValueOf(input).Elem().Kind() != reflect.Struct {

		return nil, e(
			"reason", "only structs are supported",
			"input_value_kind", reflect.ValueOf(input).Kind(),
			"input_value_elem_kind", reflect.ValueOf(input).Elem().Kind())
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		funcName := runtime.FuncForPC(reflect.ValueOf(runE).Pointer()).Name()
		e := errors.Template(funcName, errors.K.Invalid)

		in, err := SetupCmdArgs(cmd, args, reflect.TypeOf(input))
		if err != nil {
			return e(err)
		}

		// use interface{} to call fmt.Sprintf to avoid compilation error:
		//   fmt.Sprintf format %p has arg input of wrong type T
		var i interface{} = input
		if fmt.Sprintf("%p", in) != fmt.Sprintf("%p", i) {
			return e(errors.K.Internal,
				"reason", "wrong value retrieved from SetupCmdArgs",
				"in", fmt.Sprintf("%p", in),
				"input", fmt.Sprintf("%p", i))
		}

		err = runE(input)
		return e.IfNotNil(err)
	}

	err := BindCustom(cmd, f, input)
	if err != nil {
		return nil, e(err)
	}
	ConfigureHelpFuncs()
	ConfigureCommandHelp(cmd)

	return cmd, nil
}

// Binder is a utility for building trees of commands with bindings.
// Example:
//
//		 root := NewBinderC(
//				&cobra.Command{
//					Use:   "test",
//					Short: "root command",
//				}).
//				AddCommand(
//					NewBinder(
//						&testOpts{},
//						&cobra.Command{
//							Use:     "a <domains>",
//							Example: "test a x",
//						},
//						func(opts *testOpts) error {
//							return opts.run()
//						},
//						nil),
//					NewBinder(
//						&testOpts{},
//						&cobra.Command{
//							Use:     "b <domains>",
//							Example: "test b y",
//						},
//						func(opts *testOpts) error {
//							return opts.run()
//						},
//						nil))
//		if root.Error != nil {
//			panic(root.Error)
//		}
//	 ... use root.Command
type Binder struct {
	Error   error
	Command *cobra.Command
}

// NewBinderC constructs a new Binder with just a Command. This is useful for
// commands that are only parents of other commands, like the root command.
func NewBinderC(c *cobra.Command) *Binder {
	return NewBinder[any](nil, c, nil, nil)
}

// NewBinder returns a Binder initialized by running BindRunE with the given parameters
func NewBinder[T any](in *T, c *cobra.Command, runE func(*T) error, f Flagger) *Binder {
	cmd, err := BindRunE(in, c, runE, f)
	return &Binder{
		Error:   errors.ClearStacktrace(err),
		Command: cmd,
	}
}

// AddCommand adds the commands of the given Binder instances to the command of
// this Binder or append their error to the Error of this Binder if they have
// errors.
func (b *Binder) AddCommand(bound ...*Binder) *Binder {
	for _, bn := range bound {
		if bn.Error != nil {
			b.Error = errors.Append(b.Error, bn.Error)
		} else if b.Command != nil {
			b.Command.AddCommand(bn.Command)
		}
	}
	return b
}
