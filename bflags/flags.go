package bflags

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/eluv-io/errors-go"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

type cmdFlag string

const (
	flagset = "$flagset" // key used to set CmdFlags in cmd flags
	argset  = "$argset"  // key used to set argset in cmd flags
	input   = "$input"   // key used to set input in cmd flags
	cmdctx  = "$ctx"     // key used to set context in cmd flags
)

type FlagBond struct {
	isArg      bool        // false for flags, true for args
	Name       cmdFlag     // name of the flag
	Shorthand  string      // one letter short hand or the empty sting for none
	Value      interface{} // the default value (use zero value for no default)
	Usage      string      // usage string : must not be empty
	Required   bool        // true if the flag is required
	Persistent bool        // true: the flag is available to the command as well as every command under the command
	Hidden     bool        // true to set the flag as hidden
	ArgOrder   int         // for flags used to bind args
	CsvSlice   bool        // true for flags with comma separated string representation
}

var nillableKinds = []reflect.Kind{
	reflect.Chan, reflect.Func,
	reflect.Interface, reflect.Map,
	reflect.Ptr, reflect.Slice}

// isNil returns true if the given object is nil (== nil) or is a nillable type
// (channel, function, interface, map, pointer or slice) with a nil value.
func isNil(obj interface{}) bool {
	if obj == nil {
		return true
	}

	value := reflect.ValueOf(obj)
	kind := value.Kind()
	for _, k := range nillableKinds {
		if k == kind {
			return value.IsNil()
		}
	}

	return false
}

func isCsvSlice(object interface{}) bool {
	switch object.(type) {
	case
		[]string, []bool, []uint, []int,
		[]net.IP, []time.Duration:
		return true
	}
	return false
}

func (f *FlagBond) CmdString() []string {
	v := f.Value
	if isNil(v) {
		return nil
	}
	isBool := reflect.TypeOf(v) == reflect.TypeOf((*bool)(nil)) ||
		reflect.TypeOf(v) == reflect.TypeOf((**bool)(nil))
	k := reflect.ValueOf(v).Kind()
	if k == reflect.Ptr || k == reflect.Interface {
		v = reflect.ValueOf(v).Elem().Interface()
	}

	if isNil(v) {
		return nil
	}

	value := fmt.Sprintf("%v", v)
	if f.CsvSlice || isCsvSlice(v) {
		vov := reflect.ValueOf(v)
		ss := make([]string, 0, vov.Len())
		for i := 0; i < vov.Len(); i++ {
			ss = append(ss, fmt.Sprintf("%v", vov.Index(i)))
		}
		value = strings.Join(ss, ",")
	}
	if value == "" {
		return []string{}
	}

	if !f.isArg {
		if isBool {
			// pflag.flag always set the NoOptDefVal of boolean flags to 'true'
			if value == "false" {
				return []string{"--" + string(f.Name) + "=" + value}
			}
			return []string{"--" + string(f.Name)}
		}
		return []string{"--" + string(f.Name), value}
	}
	return []string{value}
}

func (f *FlagBond) Copy() *FlagBond {
	return &(*f)
}

func (f *FlagBond) SetPersistent(v bool) *FlagBond {
	f.Persistent = v
	return f
}

func (f *FlagBond) SetRequired(v bool) *FlagBond {
	f.Required = v
	return f
}

func (f *FlagBond) MarshalJSON() ([]byte, error) {
	type flagBond struct {
		Name       cmdFlag     `json:"name"`
		Shorthand  string      `json:"short_hand"`
		Value      interface{} `json:"value,omitempty"`
		Usage      string      `json:"usage"`
		Required   bool        `json:"required,omitempty"`
		Persistent bool        `json:"persistent,omitempty"`
		Hidden     bool        `json:"hidden,omitempty"`
	}
	type argBond struct {
		Name     cmdFlag     `json:"name"`
		Value    interface{} `json:"value,omitempty"`
		Usage    string      `json:"usage"`
		ArgOrder int         `json:"arg_order"`
	}
	var jsn []byte
	var err error
	if f.isArg {
		a := &argBond{
			Name:     f.Name,
			Value:    f.Value,
			Usage:    f.Usage,
			ArgOrder: f.ArgOrder,
		}
		jsn, err = json.Marshal(a)
		if err != nil {
			return nil, err
		}
	} else {
		a := &flagBond{
			Name:       f.Name,
			Shorthand:  f.Shorthand,
			Value:      f.Value,
			Usage:      f.Usage,
			Required:   f.Required,
			Persistent: f.Persistent,
			Hidden:     f.Hidden,
		}
		jsn, err = json.Marshal(a)
		if err != nil {
			return nil, err
		}
	}
	return jsn, nil
}

type CmdFlags map[cmdFlag]*FlagBond

// CmdFlags implements flag.Value in order to be stored in a hidden flag
var _ flag.Value = (CmdFlags)(nil)

func (s CmdFlags) String() string {
	bb, _ := json.Marshal(s)
	return string(bb)
}

// not intended to be called
func (s CmdFlags) Set(string) error {
	return errors.E("flagset.Set", errors.K.Invalid)
}

func (s CmdFlags) Type() string {
	return flagset
}

func (s CmdFlags) Validate() error {
	issues := make([]string, 0)
	shorts := make(map[string]cmdFlag)
	for k, v := range s {
		if v.Usage == "" {
			issues = append(issues, strings.Join([]string{"flag", string(k), "reason", "no usage"}, ","))
		}
		if v.Value == nil {
			issues = append(issues, strings.Join([]string{"flag", string(k), "reason", "no value"}, ","))
		}
		if v.Shorthand != "" {
			if o, ok := shorts[v.Shorthand]; ok {
				issues = append(issues, "duplicate shorthand for ["+string(o)+"] and ["+string(k)+"]")
			} else {
				shorts[v.Shorthand] = k
			}
		}
	}
	if len(issues) > 0 {
		return noStackTrace(errors.E("validate", errors.K.Invalid, "issues", issues))
	}
	return nil
}

func (s CmdFlags) setFor(cmd *cobra.Command) {
	setCmdFlagSet(cmd, s)
}

func (s CmdFlags) get(name cmdFlag) (*FlagBond, bool) {
	fv, ok := s[name]
	if ok {
		fv.Name = name
	}
	return fv, ok
}

func (s CmdFlags) Get(name string) (*FlagBond, bool) {
	return s.get(cmdFlag(name))
}

func (s CmdFlags) ConfigureCmd(cmd *cobra.Command, custom Flagger) error {
	if cmd == nil {
		return errors.E("configure cmd", errors.K.Invalid,
			"reason", "cmd is nil")
	}
	for k, v := range s {
		v.Name = k
		_, err := s.configureFlag(cmd, custom, v)
		if err != nil {
			return err
		}
	}
	s.setFor(cmd)
	return nil
}

// PENDING: not used so far - remove ?
func (s CmdFlags) configure(cmd *cobra.Command, name cmdFlag) (interface{}, error) {
	if cmd == nil {
		return nil, errors.E("configure flags", errors.K.Invalid,
			"reason", "cmd is nil",
			"name", name)
	}
	v, ok := s.get(name)
	if !ok {
		return nil, errors.E("configure flags", errors.K.NotExist, "name", name)
	}
	return s.configureFlag(cmd, nil, v)
}

func (s CmdFlags) configureFlag(cmd *cobra.Command, custom Flagger, v *FlagBond) (interface{}, error) {

	var pflags *flag.FlagSet
	if v.Persistent {
		pflags = cmd.PersistentFlags()
	} else {
		pflags = cmd.Flags()
	}
	flagName := string(v.Name)
	var r interface{}
	var flagged *Flagged

	if custom != nil {
		flagged = custom.Flag(v.Value)
		if flagged != nil {
			r = flagged.Ptr
			pflags.VarPF(flagged.Flag, flagName, v.Shorthand, v.Usage)
			if flagged.CsvSlice {
				v.CsvSlice = true
			}
		}
	}
	if flagged == nil {
		var err error
		r, err = s.makeFlag(pflags, v)
		if err != nil {
			return nil, err
		}
	}

	if v.Required {
		var err error
		if v.Persistent {
			err = cmd.MarkPersistentFlagRequired(flagName)
		} else {
			err = cmd.MarkFlagRequired(flagName)
		}
		if err != nil {
			return nil, err
		}
	}
	if v.Hidden {
		pflags.Lookup(flagName).Hidden = true
	}

	return r, nil
}

func (s CmdFlags) makeFlag(pflags *flag.FlagSet, v *FlagBond) (interface{}, error) {
	flagName := string(v.Name)
	var r interface{}

	switch val := v.Value.(type) {

	case string:
		r = pflags.StringP(flagName, v.Shorthand, val, v.Usage)
	case *string:
		pflags.StringVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case []string:
		r = pflags.StringSliceP(flagName, v.Shorthand, val, v.Usage)
	case *[]string:
		pflags.StringSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case **string:
		pflags.VarPF(newPtrStringValue(val), flagName, v.Shorthand, v.Usage)
		r = val

	case bool:
		r = pflags.BoolP(flagName, v.Shorthand, val, v.Usage)
	case *bool:
		pflags.BoolVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case **bool:
		fl := pflags.VarPF(newPtrBoolValue(val), flagName, v.Shorthand, v.Usage)
		// if NoOptDefVal is not defined the user has to type
		//   '--xx true' or '--xx false' (or '--xx=true' or '--xx=false')
		// if NoOptDefVal is defined the user (assuming true is the default):
		//   - CAN just type '--xx' to have it set to the default, but
		//   - HAS TO type '--xx=false' to have it set to false, and
		//   - CANNOT type '--xx true' or '--xx false'
		fl.NoOptDefVal = "true"
		fl.DefValue = "undefined - use: --" + flagName + "=false, or: --" + flagName + "=true"
		r = val
	case []bool:
		r = pflags.BoolSliceP(flagName, v.Shorthand, val, v.Usage)
	case *[]bool:
		pflags.BoolSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val

	case uint:
		r = pflags.UintP(flagName, v.Shorthand, val, v.Usage)
	case []uint:
		r = pflags.UintSliceP(flagName, v.Shorthand, val, v.Usage)
	case uint8:
		r = pflags.Uint8P(flagName, v.Shorthand, val, v.Usage)
	case uint16:
		r = pflags.Uint16P(flagName, v.Shorthand, val, v.Usage)
	case uint32:
		r = pflags.Uint32P(flagName, v.Shorthand, val, v.Usage)
	case uint64:
		r = pflags.Uint64P(flagName, v.Shorthand, val, v.Usage)
	case *uint:
		pflags.UintVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *[]uint:
		pflags.UintSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *uint8:
		pflags.Uint8VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *uint16:
		pflags.Uint16VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *uint32:
		pflags.Uint32VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *uint64:
		pflags.Uint64VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val

	case int:
		r = pflags.IntP(flagName, v.Shorthand, val, v.Usage)
	case int8:
		r = pflags.Int8P(flagName, v.Shorthand, val, v.Usage)
	case int16:
		r = pflags.Int16P(flagName, v.Shorthand, val, v.Usage)
	case int32:
		r = pflags.Int32P(flagName, v.Shorthand, val, v.Usage)
	case int64:
		r = pflags.Int64P(flagName, v.Shorthand, val, v.Usage)
	case []int:
		r = pflags.IntSliceP(flagName, v.Shorthand, val, v.Usage)
	case *int:
		pflags.IntVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case **int:
		pflags.VarPF(newPtrIntValue(val), flagName, v.Shorthand, v.Usage)
		r = val
	case *[]int:
		pflags.IntSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *int8:
		pflags.Int8VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *int16:
		pflags.Int16VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *int32:
		pflags.Int32VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case *int64:
		pflags.Int64VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case **int64:
		pflags.VarPF(newPtrInt64Value(val), flagName, v.Shorthand, v.Usage)
		r = val

	case float32:
		r = pflags.Float32P(flagName, v.Shorthand, val, v.Usage)
	case *float32:
		pflags.Float32VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val

	case float64:
		r = pflags.Float64P(flagName, v.Shorthand, val, v.Usage)
	case *float64:
		pflags.Float64VarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val

	case net.IP:
		r = pflags.IPP(flagName, v.Shorthand, val, v.Usage)
	case *net.IP:
		pflags.IPVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case []net.IP:
		r = pflags.IPSliceP(flagName, v.Shorthand, val, v.Usage)
	case *[]net.IP:
		pflags.IPSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case time.Duration:
		r = pflags.DurationP(flagName, v.Shorthand, val, v.Usage)
	case *time.Duration:
		pflags.DurationVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	case []time.Duration:
		r = pflags.DurationSliceP(flagName, v.Shorthand, val, v.Usage)
	case *[]time.Duration:
		pflags.DurationSliceVarP(val, flagName, v.Shorthand, *val, v.Usage)
		r = val
	default:
		fv, ok := reflect.ValueOf(v.Value).Elem().Interface().(flag.Value)
		if ok {
			if reflect.ValueOf(fv).Kind() == reflect.Ptr {
				// if not ptr, this might be a struct... should make a ptrValue..
				pflags.VarP(fv, flagName, v.Shorthand, v.Usage)
				r = fv
				break
			}
			//if reflect.ValueOf(fv).Kind() == reflect.Struct {
			//	pflags.VarPF(fv, flagName, v.Shorthand, v.Usage)
			//	r = fv
			//	break
			//}
		}
		return nil, errors.E("configure flags - unsupported value type",
			errors.K.NotImplemented,
			"ok", ok,
			"name", v.Name,
			"value", v.Value,
			"kind", reflect.ValueOf(fv).Kind(),
			"type", reflect.TypeOf(v.Value).String())
	}
	return r, nil
}

func (s CmdFlags) flagset(cmd *cobra.Command, name cmdFlag) (*flag.FlagSet, error) {
	if cmd == nil {
		return nil, errors.E("configure flags", errors.K.Invalid,
			"reason", "cmd is nil",
			"name", name)
	}
	v, ok := s.get(name)
	if !ok {
		return nil, errors.E("flagset", errors.K.NotExist, "name", name)
	}
	var pflags *flag.FlagSet
	if v.Persistent {
		pflags = cmd.PersistentFlags()
	} else {
		pflags = cmd.Flags()
	}
	return pflags, nil
}

func (s CmdFlags) Flagset(cmd *cobra.Command, name string) (*flag.FlagSet, error) {
	return s.flagset(cmd, cmdFlag(name))
}

func (s CmdFlags) GetString(cmd *cobra.Command, name cmdFlag) (string, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return "", err
	}
	return flagSet.GetString(string(name))
}

func (s CmdFlags) GetBool(cmd *cobra.Command, name cmdFlag) (bool, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return false, err
	}
	return flagSet.GetBool(string(name))
}

func (s CmdFlags) GetUint(cmd *cobra.Command, name cmdFlag) (uint, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetUint(string(name))
}

func (s CmdFlags) GetUint64(cmd *cobra.Command, name cmdFlag) (uint64, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetUint64(string(name))
}

func (s CmdFlags) GetInt(cmd *cobra.Command, name cmdFlag) (int, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetInt(string(name))
}

func (s CmdFlags) GetInt64(cmd *cobra.Command, name cmdFlag) (int64, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetInt64(string(name))
}

func (s CmdFlags) GetFloat32(cmd *cobra.Command, name cmdFlag) (float32, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetFloat32(string(name))
}

func (s CmdFlags) GetFloat64(cmd *cobra.Command, name cmdFlag) (float64, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetFloat64(string(name))
}

func (s CmdFlags) GetIP(cmd *cobra.Command, name cmdFlag) (net.IP, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return nil, err
	}
	return flagSet.GetIP(string(name))
}

func (s CmdFlags) GetDuration(cmd *cobra.Command, name cmdFlag) (time.Duration, error) {
	flagSet, err := s.flagset(cmd, name)
	if err != nil {
		return 0, err
	}
	return flagSet.GetDuration(string(name))
}

func noStackTrace(err error) error {
	if errx, ok := err.(*errors.Error); ok {
		_ = errx.ClearStacktrace()
	}
	return err
}

// ------- FlagSet helpers
func setCmdFlagSet(cmd *cobra.Command, s CmdFlags) {
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   flagset,
		Hidden: true,
		Value:  s,
	})
}

func GetCmdFlagSet(cmd *cobra.Command) (CmdFlags, error) {
	f := cmd.Flags().Lookup(flagset)
	if f == nil {
		return nil, errors.E("getCmdFlagSet", errors.K.NotExist)
	}
	return f.Value.(CmdFlags), nil
}

type ArgSet struct {
	Flags []*FlagBond
}

var _ flag.Value = (*ArgSet)(nil)

func (f *ArgSet) String() string {
	jsn, err := json.Marshal(f.Flags)
	if err != nil {
		panic(err)
	}
	return string(jsn)
}

// ArgUsages returns a string containing the usage information for all flags in
// the ArgSet
func (f *ArgSet) ArgUsages() string {
	if len(f.Flags) == 0 {
		return ""
	}
	sb := strings.Builder{}
	lm := 0
	for _, arg := range f.Flags {
		if len(arg.Name) > lm {
			lm = len(arg.Name)
		}
	}
	flm := fmt.Sprintf("%d", lm)
	// arg flags are expected to be correctly ordered
	for i, arg := range f.Flags {
		name := fmt.Sprintf("  %-"+flm+"s", string(arg.Name))
		sb.WriteString(name + " : " + arg.Usage)
		if i < len(f.Flags)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// not intended to be called
func (f *ArgSet) Set(string) error {
	return errors.E("argset.Set", errors.K.Invalid)
}

func (f *ArgSet) Type() string {
	return argset
}

func setCmdArgSet(cmd *cobra.Command, s []*FlagBond) {
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   argset,
		Hidden: true,
		Value:  &ArgSet{Flags: s},
	})
}

func GetCmdArgSet(cmd *cobra.Command) (*ArgSet, error) {
	f := cmd.Flags().Lookup(argset)
	if f == nil {
		return nil, errors.E("getCmdArgSet", errors.K.NotExist)
	}
	return f.Value.(*ArgSet), nil
}

// -- ptr bool value
type ptrBoolValue struct {
	p **bool
}

func newPtrBoolValue(p **bool) *ptrBoolValue {
	return &ptrBoolValue{p}
}

// IsBoolFlag is 'only' a marker func such that ptrBoolValue implements
// pflag.boolFlag (optional interface to indicate boolean flags that can be
// supplied without "=value" text) and print the 'DefValue' in help.
func (b *ptrBoolValue) IsBoolFlag() bool {
	return true
}

func (b *ptrBoolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	if err == nil {
		*b.p = &v
	}
	return err
}

func (b *ptrBoolValue) Type() string {
	return "bool"
}

func (b *ptrBoolValue) String() string {
	ret := *b.p
	if ret == nil {
		return ""
	}
	return strconv.FormatBool(*ret)
}

// -- ptr int value
type ptrIntValue struct {
	p **int
}

func newPtrIntValue(p **int) *ptrIntValue {
	return &ptrIntValue{p}
}

func (b *ptrIntValue) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 0)
	if err == nil {
		iv := int(v)
		*b.p = &iv
	}
	return err
}

func (b *ptrIntValue) Type() string {
	return "int"
}

func (b *ptrIntValue) String() string {
	ret := *b.p
	if ret == nil {
		return ""
	}
	return strconv.FormatInt(int64(*ret), 10)
}

// -- ptr int64 value
type ptrInt64Value struct {
	p **int64
}

func newPtrInt64Value(p **int64) *ptrInt64Value {
	return &ptrInt64Value{p}
}

func (b *ptrInt64Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 10, 0)
	if err == nil {
		*b.p = &v
	}
	return err
}

func (b *ptrInt64Value) Type() string {
	return "int64"
}

func (b *ptrInt64Value) String() string {
	ret := *b.p
	if ret == nil {
		return ""
	}
	return strconv.FormatInt(*ret, 10)
}

// -- ptr string value
type ptrStringValue struct {
	p **string
}

func newPtrStringValue(p **string) *ptrStringValue {
	return &ptrStringValue{p}
}

func (b *ptrStringValue) Set(s string) error {
	*b.p = &s
	return nil
}

func (b *ptrStringValue) Type() string {
	return "string"
}

func (b *ptrStringValue) String() string {
	ret := *b.p
	if ret == nil {
		return ""
	}
	return *ret
}

// inputValue implements Value in order to store any input in the flagset
type inputValue struct {
	input interface{}
}

var _ flag.Value = (*inputValue)(nil)

func (f *inputValue) String() string {
	jsn, err := json.Marshal(f)
	if err != nil {
		panic(err)
	}
	return string(jsn)
}

// not intended to be called
func (f *inputValue) Set(string) error {
	return errors.E("flagset.Set", errors.K.Invalid)
}

func (f *inputValue) Type() string {
	return input
}

func setCmdInput(cmd *cobra.Command, v interface{}) {
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   input,
		Hidden: true,
		Value:  &inputValue{input: v},
	})
}

func GetCmdInput(cmd *cobra.Command) (interface{}, bool) {
	f := cmd.Flags().Lookup(input)
	if f == nil {
		return nil, false
	}
	return f.Value.(*inputValue).input, true
}

// ctxValue implements Value in order to store any context in the flagset
type ctxValue struct {
	typ string
	ctx interface{}
}

var _ flag.Value = (*ctxValue)(nil)

func (f *ctxValue) String() string {
	jsn, err := json.Marshal(f.ctx)
	if err != nil {
		panic(err)
	}
	return string(jsn)
}

// not intended to be called
func (f *ctxValue) Set(string) error {
	return errors.E("flags.Set", errors.K.Invalid)
}

func (f *ctxValue) Type() string {
	return f.typ
}

func SetCmdCtx(cmd *cobra.Command, v interface{}) {
	f := cmd.Flags().Lookup(cmdctx)
	if f != nil && f.Value.(*ctxValue).ctx == v {
		return
	}
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   cmdctx,
		Hidden: true,
		Value:  &ctxValue{typ: cmdctx, ctx: v},
	})
}

func GetCmdCtx(cmd *cobra.Command) (interface{}, bool) {
	if cmd == nil {
		return nil, false
	}
	f := cmd.Flags().Lookup(cmdctx)
	if f == nil {
		return nil, false
	}
	return f.Value.(*ctxValue).ctx, true
}

func AddToCmdCtx(cmd *cobra.Command, name string, v interface{}) bool {
	f := cmd.Flags().Lookup(name)
	if f != nil {
		return false
	}
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   name,
		Hidden: true,
		Value:  &ctxValue{typ: name, ctx: v},
	})
	return true
}

func GetFromCmdCtx(cmd *cobra.Command, name string) (interface{}, bool) {
	if cmd == nil {
		return nil, false
	}
	f := cmd.Flags().Lookup(name)
	if f == nil {
		return nil, false
	}
	return f.Value.(*ctxValue).ctx, true
}
