package app

import (
	"encoding/json"
	"reflect"

	"github.com/eluv-io/errors-go"
	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/bflags"
)

type CmdFlag interface {
	Value() interface{}
	CmdPath() string
	FlagName() string
	Arg() bool
}

type FlagsValidator interface {
	Validate(t reflect.Type, flags []CmdFlag) error
}

// FlagsDictionary enables normalization of flag names vs the type they represent.
//
// Constructing the flags dictionary has to happen after flags have been bound
// to the commands input via `bflags.Bind`.
// In the case of an app, this might be done after having called the Cobra func:
// ```
//     a, _ := app.NewApp(spec, nil)
//     root, err := a.Cobra()
//	   ..
// ```
// Steps:
// - construct an new FlagsDictionary passing a [possibly nil] validator
// - call Validate(root):
//   - if no validator was provided this just loads the dictionary
//   - otherwise it calls the validator for each type found in flags values
// - call String() to have a json of the dictionary (for 'manual' inspection).
//
type FlagsDictionary interface {
	String() string
	Validate(root *cobra.Command) error
}

func NewFlagDictionary(v FlagsValidator, withArgs bool) FlagsDictionary {
	return &flagDictionary{
		validator: v,
		withArgs:  withArgs,
		flags:     make(map[string][]CmdFlag),
	}
}

//
// ----- impl -----
//

type cmdFlag struct {
	value    interface{}
	cmdPath  string
	flagName string
	isArg    bool
}

func (c *cmdFlag) MarshalJSON() ([]byte, error) {
	type cf struct {
		Type     string `json:"type"`
		CmdPath  string `json:"cmd_path"`
		FlagName string `json:"flag_name"`
		Arg      bool   `json:"arg"`
	}
	t := reflect.TypeOf(c.value)
	f := &cf{Type: t.String(), CmdPath: c.cmdPath, FlagName: c.flagName, Arg: c.isArg}
	return json.Marshal(f)
}

func (c *cmdFlag) CmdPath() string {
	return c.cmdPath
}

func (c *cmdFlag) FlagName() string {
	return c.flagName
}

func (c *cmdFlag) Value() interface{} {
	return c.value
}

func (c *cmdFlag) Arg() bool {
	return c.isArg
}

type flagDictionary struct {
	validator FlagsValidator
	withArgs  bool
	flags     map[string][]CmdFlag
}

func (f *flagDictionary) String() string {
	bb, err := json.MarshalIndent(f.flags, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(bb)
}

func (f *flagDictionary) Validate(root *cobra.Command) error {
	err := f.inspect(root, "")
	if err != nil {
		return err
	}
	if f.validator == nil {
		return nil
	}
	for _, flags := range f.flags {
		typ := reflect.TypeOf(flags[0].Value())
		e := f.validator.Validate(typ, flags)
		if e != nil {
			return e
		}
	}
	return nil
}

func (f *flagDictionary) inspectFlag(name string, flag *bflags.FlagBond, cmdPath string, arg bool) {
	t := reflect.TypeOf(flag.Value)
	tn := t.String()
	flags, ok := f.flags[tn]
	if !ok {
		flags = make([]CmdFlag, 0)
		f.flags[tn] = flags
	}
	flags = append(flags, &cmdFlag{
		value:    flag.Value,
		cmdPath:  cmdPath,
		flagName: name,
		isArg:    arg})
	f.flags[tn] = flags
}

func (f *flagDictionary) inspectFlags(flags bflags.CmdFlags, cmdPath string) {
	for k, fl := range flags {
		f.inspectFlag(string(k), fl, cmdPath, false)
	}
}

func (f *flagDictionary) inspectArgs(flags []*bflags.FlagBond, cmdPath string) {
	for _, fl := range flags {
		f.inspectFlag(string(fl.Name), fl, cmdPath, true)
	}
}

func (f *flagDictionary) inspect(c *cobra.Command, cmdPath string) error {
	flags, err := bflags.GetCmdFlagSet(c)
	if err != nil && !errors.IsNotExist(err) {
		return err
	}
	argset, err := bflags.GetCmdArgSet(c)
	if err != nil && !errors.IsNotExist(err) {
		return err
	}
	if cmdPath != "" {
		cmdPath += "/"
	}
	cmdPath += c.Name()

	var args []*bflags.FlagBond
	if argset != nil {
		args = argset.Flags
	}
	f.inspectFlags(flags, cmdPath)
	f.inspectArgs(args, cmdPath)

	for _, child := range c.Commands() {
		err = f.inspect(child, cmdPath)
		if err != nil {
			return err
		}
	}
	return nil
}
