package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/eluv-io/ecobra-go/bflags"
	"github.com/eluv-io/errors-go"
)

func SpecOf(cmd *cobra.Command) *spec {
	f := cmd.Flags().Lookup(SpecKey)
	if f == nil {
		return nil
	}
	s, ok := f.Value.(*spec)
	if !ok {
		return nil
	}
	return s
}

var _ = SpecOf

// ----- Runtime -----

// Ctor is the type of function creating an 'input' parameter for a command
// The returned value will be initialized with the default values found in the
// App spec.
type Ctor func() interface{}

// Runfn is the type of function that a command shall execute at runtime. These
// functions are expected to conform to the following prototype:
//
//	func(ctx *CmdCtx, in interface{}) (interface{}, error)
//
// - At least one input parameter:
//   - first of type *CmdCtx
//   - optionally a second parameter that is the input of the command
//
// - 2 output parameters, the last one being an error
type Runfn interface{}

type Runtime struct {
	cobraFns map[string]CobraFunction
	inputs   map[string]Ctor
	runFns   map[string]interface{}
}

func isRunFn(name string, fn interface{}) error {
	e := errors.Template("rt function")
	f := reflect.ValueOf(fn)
	if f.Kind() != reflect.Func {
		return e("reason", "not a function", "function", name)
	}
	typ := f.Type()
	if typ.NumIn() < 1 {
		return e("reason", "need at least 1 parameters",
			"function", name)
	}
	if typ.In(0) != reflect.TypeOf((*CmdCtx)(nil)) {
		return e("reason", "first parameter must be *CmdCtx",
			"parameter_type", typ.In(0).String(),
			"function", name)
	}
	outCount := typ.NumOut()
	if outCount < 1 || outCount > 2 {
		return e("reason", "need 1 or 2 return values",
			"return count", outCount,
			"function", name)
	}
	last := 1
	if outCount == 1 {
		last = 0
	}
	if !typ.Out(last).AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
		return e("reason", "last return values must be error",
			"last return value", typ.Out(last).Name(),
			"function", name)
	}
	return nil
}

// inputCtor returns the input of a command. If input is
//   - a function, it must be a constructor (func without parameter and a single
//     return) the returned value of the function is returned as the input.
//   - otherwise input is returned unchanged
func inputCtor(input interface{}) (interface{}, error) {
	e := errors.Template("input ctor", errors.K.Invalid)
	fn := reflect.ValueOf(input)
	if fn.Kind() != reflect.Func {
		// not a function, return as is
		return input, nil
	}
	typ := fn.Type()
	if typ.NumIn() != 0 {
		return nil, e("reason", "need no parameter")
	}
	outCount := typ.NumOut()
	if outCount != 1 {
		return nil, e("reason", "need 1 return value",
			"return count", outCount)
	}

	res := fn.Call([]reflect.Value{})
	return res[0].Interface(), nil
}

func RtFunctions(
	cobraFns map[string]CobraFunction,
	inputs map[string]Ctor,
	runFns map[string]Runfn) (*Runtime, error) {

	if cobraFns == nil {
		cobraFns = make(map[string]CobraFunction)
	}
	if inputs == nil {
		inputs = make(map[string]Ctor)
	}
	rt := &Runtime{
		cobraFns: cobraFns,
		inputs:   inputs,
		runFns:   make(map[string]interface{}),
	}
	e := errors.Template("rt functions")
	for name, fn := range runFns {
		if err := isRunFn(name, fn); err != nil {
			return nil, e(err)
		}
		rt.runFns[name] = fn
	}
	return rt, nil
}

// ----- spec -----

// SpecKey is the context key for the spec
const SpecKey = "$spec"

type spec struct {
	Categories []*CmdCategory `json:"categories"`
	CmdRoot    *Cmd           `json:"cmd_root"`
}

func NewSpec(categories []*CmdCategory, cmdRoot *Cmd) *spec {
	return &spec{
		Categories: categories,
		CmdRoot:    cmdRoot,
	}
}

func (s *spec) setFor(cmd *cobra.Command) {
	cmd.Flags().AddFlag(&flag.Flag{
		Name:   SpecKey,
		Hidden: true,
		Value:  s,
	})
}

// spec implements flag.Value in order to be stored in a hidden flag
var _ flag.Value = (*spec)(nil)

func (s *spec) String() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(s)
	if err != nil {
		panic("Failed to marshal json: " + err.Error())
	}
	return string(buf.Bytes())
}

// Set is needed to satisfy the Value interface but is not intended to be called
func (s *spec) Set(string) error {
	return errors.E("spec.Set", errors.K.Invalid)
}

func (s *spec) Type() string {
	return SpecKey
}

// ----- App -----

type CobraFunction func(cmd *cobra.Command, args []string) error
type CommandStart func(cmd *cobra.Command, flagsAndArgs map[string]string, in interface{})
type CommandEnd func(cmd *cobra.Command, out interface{}, err error)

type App struct {
	spec          *spec
	root          *cobra.Command
	rt            *Runtime
	customFlags   bflags.Flagger // flag support for specific types
	flagsChecker  CobraFunction  // support for flags checking before command run
	cmdStart      CommandStart   // cmdStart is invoked immediately before the command runs
	cmdEnd        CommandEnd     // cmdEnd is invoked after the command ran
	results       []*CmdResult   // monitored results
	printResultFn PrintResultFn  // user provided func to print results (default is used if nil)
}

func NewApp(spec *spec, rtSpec *Runtime) (*App, error) {
	e := errors.Template("new app", errors.K.Invalid)
	if spec == nil {
		return nil, e("reason", "nil spec")
	}
	if rtSpec == nil {
		rtSpec = &Runtime{
			cobraFns: make(map[string]CobraFunction),
			inputs:   make(map[string]Ctor),
			runFns:   make(map[string]interface{}),
		}
	}
	if spec.CmdRoot == nil {
		return nil, e("reason", "no root command")
	}
	a := &App{
		spec: spec,
		rt:   rtSpec,
	}
	a.spec.CmdRoot.app = a
	return a, nil
}

func NewAppFromSpec(jsonSpec string, rtSpec *Runtime) (*App, error) {
	s, err := readSpec(jsonSpec)
	if err != nil {
		return nil, err
	}
	return NewApp(s, rtSpec)
}

func (a *App) WithCustomFlags(custom bflags.Flagger) *App {
	a.customFlags = custom
	return a
}

func (a *App) WithFlagsChecker(flagsChecker CobraFunction) *App {
	a.flagsChecker = flagsChecker
	return a
}

func readSpec(jspec string) (*spec, error) {
	spec := &spec{}
	if err := json.Unmarshal([]byte(jspec), spec); err != nil {
		return nil, err
	}
	return spec, nil
}

func (a *App) Spec() *spec {
	return a.spec
}

func (a *App) Cobra() (*cobra.Command, error) {
	if a.root == nil {
		r, err := a.spec.CmdRoot.ToCobra(nil, a.customFlags)
		if err != nil {
			return nil, err
		}
		a.spec.setFor(r)
		a.root = r
		a.configureHelp()
	}
	return a.root, nil
}

func (a *App) NewCobra() (*cobra.Command, error) {
	a.root = nil
	return a.Cobra()
}

func (a *App) configureHelp() {
	// configure categories and template functions
	if len(a.spec.Categories) > 0 {
		AddTemplateFunc("categories",
			func(cmdRoot *cobra.Command) []*CmdCategory {
				return NewCategories(a.spec.Categories, cmdRoot).GetCategories()
			})
	} else {
		AddTemplateFunc("categories", func() string { return "" })
	}
	bflags.ConfigureHelpFuncs()

	// configure help
	configureHelp(a.root)
}

func (a *App) Command(path ...string) (*Cmd, error) {
	e := errors.Template("Command", errors.K.Invalid, "path", strings.Join(path, ","))
	if len(path) == 0 {
		return a.spec.CmdRoot, nil
	}
	if path[0] != a.spec.CmdRoot.Name() {
		return nil, e(errors.K.NotExist)
	}
	if len(path) == 1 {
		return a.spec.CmdRoot, nil
	}
	return a.spec.CmdRoot.Sub(path[1:])
}

func (a *App) SetCommandStart(cmdStart CommandStart) {
	a.cmdStart = cmdStart
}

func (a *App) SetCommandEnd(cmdEnd CommandEnd) {
	a.cmdEnd = cmdEnd
}

func (a *App) onExit() {
	a.printResults("exit signal")
}

func (a *App) getResults() []*CmdResult {
	return a.results
}

func (a *App) printResults(reason string) {
	if a.printResultFn != nil {
		a.printResultFn(a.results)
		return
	}
	if len(a.results) == 0 {
		return
	}
	fmt.Println("\n" + reason + " - intermediary results") // avoid ^Cxx stick in front of result
	for _, r := range a.results {
		fmt.Println(r.String())
	}
	fmt.Println()
}

func (a *App) SetMonitorResults(b bool) {
	if b {
		a.results = make([]*CmdResult, 0)
		registerSignalHandler(a.onExit, false)
	} else {
		a.results = nil
	}
}

func (a *App) retrieveContext(cmd *cobra.Command) *CmdCtx {
	var ctx *CmdCtx
	if c, ok := bflags.GetCmdCtx(cmd); ok {
		ctx, ok = c.(*CmdCtx)
	}
	if ctx == nil {
		ctx = NewCmdCtx()
	}
	return ctx
}

func (a *App) runStub(fn interface{}, name string) CobraFunction {

	return func(cmd *cobra.Command, args []string) (err error) {
		e := errors.Template("command failed", errors.K.Invalid, "cmd", cmd.Name(),
			"args", "["+strings.Join(args, ",")+"]")
		defer func() {
			if r := recover(); r != nil {
				err = e("cause", r)
			}
		}()
		ctx := a.retrieveContext(cmd)
		if a.results != nil {
			// if result monitoring is enabled make sure the add result function
			// is on the cmdCtx
			var arfn AddResultFn = func(key string, out interface{}, err error) {
				a.results = append(a.results, newCommandResult(key, out, err))
			}
			ctx.Set(CtxAddResultFn, arfn)
			ctx.Set(CtxPrintResultFn, a.printResults)
			ctx.Set(CtxGetResultFn, a.getResults)
		}
		m, err := bflags.SetArgs(cmd, args)
		if err != nil {
			return e(err, "reason", "error retrieving flag, arg or input")
		}
		if a.flagsChecker != nil {
			err = a.flagsChecker(cmd, args)
			if err != nil {
				return e(err, "reason", "flags check failure")
			}
		}

		f := reflect.ValueOf(fn)
		if f.Kind() != reflect.Func {
			return e(err, "reason", "not a function", "name", name, "fn", fn)
		}
		if name == "" {
			name = runtime.FuncForPC(f.Pointer()).Name()
		}
		if err := isRunFn(name, fn); err != nil {
			return e(err)
		}
		if a.cmdStart != nil {
			a.cmdStart(cmd, bflags.GetFlagArgSet(cmd), m)
		}

		res, err := a.callFn(name, f, ctx, m)
		if err != nil {
			// definition of function to call is invalid or panic'ed
			return e(err)
		}

		// now, unwrap result of function call
		outCount := len(res)
		if outCount < 1 || outCount > 2 {
			return e("reason", "expected 1 or 2 returned values",
				"returned values", outCount)
		}
		var out interface{}
		last := 0
		if outCount == 2 {
			out = res[0].Interface()
			last = 1
		}
		if r, ok := res[last].Interface().(error); ok && !reflect.ValueOf(r).IsNil() {
			err = r
		}
		if a.cmdEnd != nil {
			defer a.cmdEnd(cmd, out, err)
		}
		if err != nil {
			return e(err)
		}
		ctx.Set(CtxResult, out)
		return nil
	}

}

func (a *App) callFn(name string, fn reflect.Value, params ...interface{}) (v []reflect.Value, err error) {
	e := errors.Template("callFn", errors.K.Invalid, "name", name, "params", params)

	defer func() {
		if r := recover(); r != nil {
			err = e("cause", r, "note", "please verify parameters of run function")
		}
	}()
	if len(params) != fn.Type().NumIn() {
		if len(params) > fn.Type().NumIn() {
			// mainly for commands without other parameter than cmdCtx: the
			// passed in second parameter is nil
			ok := true
			for _, p := range params[fn.Type().NumIn():] {
				if p != nil {
					ok = false
					break
				}
			}
			if ok {
				params = params[:fn.Type().NumIn()]
			}
		}
		if len(params) != fn.Type().NumIn() {
			return nil, e("reason", "invalid parameters count",
				"expected params", fn.Type().NumIn(),
				"actual params", len(params))
		}
	}
	//log.Debug("calling " + name)

	in := make([]reflect.Value, len(params))
	for k, param := range params {
		in[k] = reflect.ValueOf(param)
	}
	return fn.Call(in), nil
}

// ----- Cmd -----

// ValidatorCtor is a constructor validator
type ValidatorCtor func(c *cobra.Command) func(c *cobra.Command, args []string) error

type CobraFunc struct {
	name string
	fn   CobraFunction
}

func CobraFnWithName(name string) CobraFunc {
	return CobraFunc{name: name}
}

func CobraFn(fn CobraFunction) CobraFunc {
	return CobraFunc{fn: fn}
}

func (c *CobraFunc) String() string {
	ret, _ := c.MarshalText()
	return string(ret)
}

// MarshalText implements custom marshaling using the string representation.
func (c *CobraFunc) MarshalText() ([]byte, error) {
	n := c.name
	if n == "" && c.fn != nil {
		n = runtime.FuncForPC(reflect.ValueOf(c.fn).Pointer()).Name()
	}
	return []byte(n), nil
}

// UnmarshalText implements custom unmarshaling from the string representation.
func (c *CobraFunc) UnmarshalText(text []byte) error {
	c.name = string(text)
	return nil
}

type RunFunc struct {
	name string
	fn   Runfn
}

func (c RunFunc) IsNil() bool {
	return c.name == "" && c.fn == nil
}

func (c RunFunc) Func() Runfn {
	return c.fn
}

func RunFnWithName(name string) RunFunc {
	return RunFunc{name: name}
}

func RunFn(fn Runfn) RunFunc {
	return RunFunc{fn: fn}
}

func (c *RunFunc) String() string {
	ret, _ := c.MarshalText()
	return string(ret)
}

// MarshalText implements custom marshaling using the string representation.
func (c *RunFunc) MarshalText() ([]byte, error) {
	n := c.name
	if n == "" && c.fn != nil {
		n = runtime.FuncForPC(reflect.ValueOf(c.fn).Pointer()).Name()
	}
	return []byte(n), nil
}

// UnmarshalText implements custom unmarshaling from the string representation.
func (c *RunFunc) UnmarshalText(text []byte) error {
	c.name = string(text)
	return nil
}

type mstring string

func (m *mstring) UnmarshalJSON(data []byte) error {
	s := new(string)
	err := json.Unmarshal(data, s)
	if err == nil {
		*m = mstring(*s)
	}
	sb := make([]string, 0)
	err = json.Unmarshal(data, &sb)
	if err != nil {
		return err
	}
	*m = mstring(strings.Join(sb, "\n"))
	return nil
}

func (m *mstring) MarshalJSON() ([]byte, error) {
	s := string(*m)
	ss := strings.Split(s, "\n")
	if len(ss) <= 1 {
		return json.Marshal(s)
	}
	return json.Marshal(ss)
}

type CmdInput interface{}
type Cmd struct {
	app                        *App
	Use                        string            `json:"use"`
	Aliases                    []string          `json:"aliases,omitempty"`
	SuggestFor                 []string          `json:"suggest_for,omitempty"`
	Short                      string            `json:"short"`
	Long                       mstring           `json:"long"`
	Category                   string            `json:"category"`
	Example                    mstring           `json:"example"`
	ValidArgs                  []string          `json:"valid_args,omitempty"`
	Args                       string            `json:"args,omitempty"`
	ArgsValidator              ValidatorCtor     `json:"-"` // additional validator
	ArgAliases                 []string          `json:"arg_aliases,omitempty"`
	BashCompletionFunction     string            `json:"bash_completion_function,omitempty"`
	Deprecated                 string            `json:"deprecated,omitempty"`
	Hidden                     bool              `json:"hidden,omitempty"`
	Annotations                map[string]string `json:"annotations,omitempty"`
	Version                    string            `json:"version,omitempty"`
	PersistentPreRunE          CobraFunc         `json:"persistent_pre_run_e,omitempty"`
	PreRunE                    CobraFunc         `json:"pre_run_e,omitempty"`
	RunE                       RunFunc           `json:"run_e"`
	PostRunE                   CobraFunc         `json:"post_run_e,omitempty"`
	PersistentPostRunE         CobraFunc         `json:"persistent_post_run_e,omitempty"`
	SilenceErrors              bool              `json:"silence_errors,omitempty"`
	SilenceUsage               bool              `json:"silence_usage,omitempty"`
	DisableFlagParsing         bool              `json:"disable_flag_parsing,omitempty"`
	DisableAutoGenTag          bool              `json:"disable_auto_gen_tag,omitempty"`
	DisableFlagsInUseLine      bool              `json:"disable_flags_in_use_line,omitempty"`
	DisableSuggestions         bool              `json:"disable_suggestions,omitempty"`
	SuggestionsMinimumDistance int               `json:"suggestions_minimum_distance,omitempty"`
	TraverseChildren           bool              `json:"traverse_children,omitempty"`
	InputCtor                  string            `json:"input_ctor"`             // name of input in app's map
	Input                      CmdInput          `json:"input,omitempty"`        // json of input or input object
	SubCommands                []*Cmd            `json:"sub_commands,omitempty"` // sub commands
}

func (c *Cmd) Name() string {
	name := c.Use
	i := strings.Index(name, " ")
	if i >= 0 {
		name = name[:i]
	}
	return name
}

func (c *Cmd) UpdateExamples(upd func(s string) string) {
	if upd == nil {
		return
	}
	c.Example = mstring(upd(string(c.Example)))
	for _, child := range c.SubCommands {
		child.UpdateExamples(upd)
	}
}

func (c *Cmd) Sub(path []string) (*Cmd, error) {
	e := errors.Template("Sub", errors.K.Invalid, "path", strings.Join(path, ","))
	if len(path) == 0 {
		return nil, e("reason", "empty path")
	}
	for _, sub := range c.SubCommands {
		if path[0] == sub.Name() {
			if len(path) == 1 {
				return sub, nil
			}
			return sub.Sub(path[1:])
		}
	}
	return nil, e(errors.K.NotExist)
}

func (c *Cmd) cobraFn(cf CobraFunc) CobraFunction {
	if cf.fn != nil {
		return cf.fn
	}
	if cf.name == "" {
		return nil
	}
	fn, ok := c.app.rt.cobraFns[cf.name]
	if !ok {
		panic(fmt.Sprintf("Function [%s] not found", cf.name))
	}
	return fn
}

func (c *Cmd) persistentPreRunE(parent *cobra.Command, cf CobraFunc) CobraFunction {
	res := c.cobraFn(cf)
	if parent == nil {
		// if parent is nil we're setting up the root
		f := func(cmd *cobra.Command, args []string) error {
			// the context might have been set on root
			root := cmd.Root()
			var ctx *CmdCtx
			c, ok := bflags.GetCmdCtx(root)
			if ok {
				ctx, _ = c.(*CmdCtx)
			}
			if ctx == nil {
				ctx = NewCmdCtx()
			}
			ctx.Set(CtxCmd, cmd)
			bflags.SetCmdCtx(cmd, ctx)
			if cmd != root && !ok {
				// also set it on root
				bflags.SetCmdCtx(root, ctx)
			}
			if res == nil {
				return nil
			}
			return res(cmd, args)
		}
		return f
	}
	return res
}

func (c *Cmd) runFn(fn RunFunc) (CobraFunction, error) {
	f := fn.fn
	if f == nil {
		if fn.name == "" {
			return nil, nil
		}
		ok := false
		f, ok = c.app.rt.runFns[fn.name]
		if !ok {
			return nil, errors.E("runFn", errors.K.NotExist, "function", fn.name)
		}
	}
	return c.app.runStub(f, fn.name), nil
}

func (c *Cmd) decodeInput() (interface{}, error) {
	e := errors.Template("make input")
	if c.InputCtor == "" {
		return inputCtor(c.Input)
	}
	ctor, ok := c.app.rt.inputs[c.InputCtor]
	if !ok {
		return nil, e(errors.K.NotExist, "reason", "input not found", "input", c.InputCtor)
	}
	in := ctor()
	if c.Input != nil {

		cfg := &mapstructure.DecoderConfig{
			TagName: "json",
			Result:  in,
		}
		decoder, err := mapstructure.NewDecoder(cfg)
		if err != nil {
			return nil, err
		}
		err = decoder.Decode(c.Input)
		if err != nil {
			return nil, err
		}
	}
	return in, nil
}

func parsePositional(s string) (string, int, int, error) {
	e := errors.Template("parse positional", "args", s)
	if s == "" {
		return "", 0, 0, nil
	}
	s = strings.Trim(s, " ")
	ndx := strings.Index(s, "(")
	if ndx < 0 {
		return s, 0, 0, nil
	}
	edx := strings.Index(s, ")")
	if edx < 0 {
		return "", 0, 0, e("reason", "unclosed (")
	}
	name := s[0:ndx]
	val := s[ndx+1 : edx]
	cx := strings.Index(val, ",")
	if cx < 0 {
		n, err := strconv.Atoi(val)
		if err != nil {
			return "", 0, 0, e(err, "converting val", val)
		}
		return name, n, 0, nil
	}
	n, err := strconv.Atoi(val[:cx])
	if err != nil {
		return "", 0, 0, e(err, "converting val", val[:cx])
	}
	m, err := strconv.Atoi(strings.TrimSpace(val[cx+1:]))
	if err != nil {
		return "", 0, 0, e(err, "converting val", val[cx+1:])
	}
	return name, n, m, nil
}

func (c *Cmd) ToCobra(parent *cobra.Command, f bflags.Flagger) (*cobra.Command, error) {

	e := errors.Template("to_cobra")
	var positional cobra.PositionalArgs
	pos, n, m, err := parsePositional(c.Args)
	if err != nil {
		return nil, e(err)
	}
	switch pos {
	case "NoArgs":
		positional = cobra.NoArgs
	case "OnlyValidArgs":
		positional = cobra.OnlyValidArgs
	case "ArbitraryArgs":
		positional = cobra.ArbitraryArgs
	case "MinimumNArgs":
		positional = cobra.MinimumNArgs(n)
	case "MaximumNArgs":
		positional = cobra.MaximumNArgs(n)
	case "ExactArgs":
		positional = cobra.ExactArgs(n)
	case "RangeArgs":
		positional = cobra.RangeArgs(n, m)
	case "":
	default:
		return nil, e("reason", "unknown positional function",
			"positional function", pos)
	}
	runE, err := c.runFn(c.RunE)
	if err != nil {
		return nil, e(err)
	}

	cmd := &cobra.Command{
		Use:                        c.Use,
		Aliases:                    c.Aliases,
		SuggestFor:                 c.SuggestFor,
		Short:                      c.Short,
		Long:                       string(c.Long),
		Example:                    string(c.Example),
		ValidArgs:                  c.ValidArgs,
		ArgAliases:                 c.ArgAliases,
		Args:                       positional,
		BashCompletionFunction:     c.BashCompletionFunction,
		Deprecated:                 c.Deprecated,
		Hidden:                     c.Hidden,
		Annotations:                c.Annotations,
		Version:                    c.Version,
		PersistentPreRunE:          c.persistentPreRunE(parent, c.PersistentPreRunE),
		PreRunE:                    c.cobraFn(c.PreRunE),
		RunE:                       runE,
		PostRunE:                   c.cobraFn(c.PostRunE),
		PersistentPostRunE:         c.cobraFn(c.PersistentPostRunE),
		SilenceErrors:              c.SilenceErrors,
		SilenceUsage:               c.SilenceUsage,
		DisableFlagParsing:         c.DisableFlagParsing,
		DisableAutoGenTag:          c.DisableAutoGenTag,
		DisableFlagsInUseLine:      c.DisableFlagsInUseLine,
		DisableSuggestions:         c.DisableSuggestions,
		SuggestionsMinimumDistance: c.SuggestionsMinimumDistance,
		TraverseChildren:           c.TraverseChildren,
	}
	if c.Category != "" {
		annotateCmdCategory(cmd, c.Category)
	}
	var in interface{}
	in, err = c.decodeInput()
	if err == nil {
		if parent != nil {
			parent.AddCommand(cmd)
		}
		err = bflags.BindCustom(cmd, f, in)
	}
	if err != nil {
		return nil, err
	}
	if c.ArgsValidator != nil {
		// additional 'positional' function that can do further validation
		cmd.Args = c.ArgsValidator(cmd)
	}

	for _, sub := range c.SubCommands {
		sub.app = c.app
		_, err := sub.ToCobra(cmd, f)
		if err != nil {
			return nil, err
		}
	}
	return cmd, nil
}

type JCmd struct {
	app                        *App
	Use                        string            `json:"use"`
	Aliases                    []string          `json:"aliases,omitempty"`
	SuggestFor                 []string          `json:"suggest_for,omitempty"`
	Short                      string            `json:"short"`
	Long                       mstring           `json:"long,omitempty"`
	Category                   string            `json:"category,omitempty"`
	Example                    mstring           `json:"example,omitempty"`
	ValidArgs                  []string          `json:"valid_args,omitempty"`
	Args                       string            `json:"args,omitempty"`
	ArgsValidator              ValidatorCtor     `json:"-"` // additional validator
	ArgAliases                 []string          `json:"arg_aliases,omitempty"`
	BashCompletionFunction     string            `json:"bash_completion_function,omitempty"`
	Deprecated                 string            `json:"deprecated,omitempty"`
	Hidden                     bool              `json:"hidden,omitempty"`
	Annotations                map[string]string `json:"annotations,omitempty"`
	Version                    string            `json:"version,omitempty"`
	PersistentPreRunE          string            `json:"persistent_pre_run_e,omitempty"`
	PreRunE                    string            `json:"pre_run_e,omitempty"`
	RunE                       string            `json:"run_e,omitempty"`
	PostRunE                   string            `json:"post_run_e,omitempty"`
	PersistentPostRunE         string            `json:"persistent_post_run_e,omitempty"`
	SilenceErrors              bool              `json:"silence_errors,omitempty"`
	SilenceUsage               bool              `json:"silence_usage,omitempty"`
	DisableFlagParsing         bool              `json:"disable_flag_parsing,omitempty"`
	DisableAutoGenTag          bool              `json:"disable_auto_gen_tag,omitempty"`
	DisableFlagsInUseLine      bool              `json:"disable_flags_in_use_line,omitempty"`
	DisableSuggestions         bool              `json:"disable_suggestions,omitempty"`
	SuggestionsMinimumDistance int               `json:"suggestions_minimum_distance,omitempty"`
	TraverseChildren           bool              `json:"traverse_children,omitempty"`
	InputCtor                  string            `json:"input_ctor,omitempty"`   // name of input in app's map
	Input                      CmdInput          `json:"input,omitempty"`        // json of input or input object
	SubCommands                []*Cmd            `json:"sub_commands,omitempty"` // sub commands
}

func (c *Cmd) MarshalJSON() ([]byte, error) {
	jc := JCmd{
		app:                        c.app,
		Use:                        c.Use,
		Aliases:                    c.Aliases,
		SuggestFor:                 c.SuggestFor,
		Short:                      c.Short,
		Long:                       c.Long,
		Category:                   c.Category,
		Example:                    c.Example,
		ValidArgs:                  c.ValidArgs,
		Args:                       c.Args,
		ArgsValidator:              c.ArgsValidator,
		ArgAliases:                 c.ArgAliases,
		BashCompletionFunction:     c.BashCompletionFunction,
		Deprecated:                 c.Deprecated,
		Hidden:                     c.Hidden,
		Annotations:                c.Annotations,
		Version:                    c.Version,
		PersistentPreRunE:          c.PersistentPreRunE.String(),
		PreRunE:                    c.PreRunE.String(),
		RunE:                       c.RunE.String(),
		PostRunE:                   c.PostRunE.String(),
		PersistentPostRunE:         c.PersistentPostRunE.String(),
		SilenceErrors:              c.SilenceErrors,
		SilenceUsage:               c.SilenceUsage,
		DisableFlagParsing:         c.DisableFlagParsing,
		DisableAutoGenTag:          c.DisableAutoGenTag,
		DisableFlagsInUseLine:      c.DisableFlagsInUseLine,
		DisableSuggestions:         c.DisableSuggestions,
		SuggestionsMinimumDistance: c.SuggestionsMinimumDistance,
		TraverseChildren:           c.TraverseChildren,
		InputCtor:                  c.InputCtor,
		Input:                      c.Input,
		SubCommands:                c.SubCommands,
	}
	ti := reflect.TypeOf(jc.Input)
	if ti != nil && ti.Kind() == reflect.Func {
		jc.Input = runtime.FuncForPC(reflect.ValueOf(jc.Input).Pointer()).Name()
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	err := enc.Encode(&jc)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// AddResultFn and PrintResultFn are function for results book keeping
type AddResultFn func(key string, out interface{}, err error)
type PrintResultFn func(results []*CmdResult)

type CmdResult struct {
	Key    string      `json:"key"`
	Result interface{} `json:"result,omitempty"`
	Error  string      `json:"error,omitempty"`
}

func newCommandResult(key string, out interface{}, err error) *CmdResult {
	serr := ""
	if err != nil {
		serr = fmt.Sprintf("%v", err)
	}
	return &CmdResult{
		Key:    key,
		Result: out,
		Error:  serr,
	}
}

func (r *CmdResult) String() string {
	res := "no result"
	if r.Result != nil {
		switch rt := r.Result.(type) {
		default:
			//json also works for simple types:
			//bool, int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64
			bb, err := json.Marshal(r.Result)
			if err == nil {
				res = string(bb)
			} else {
				res = fmt.Sprintf("%v", r.Result)
			}
		case *string:
			res = *rt
		case string:
			res = rt
		}
	}
	ret := fmt.Sprintf("%s: %s", r.Key, res)

	serr := ""
	if len(r.Error) > 0 {
		serr = fmt.Sprintf("error: %v", r.Error)
	}
	if serr != "" {
		ret += "\n" + serr
	}
	return ret
}
