# ecobra

Module `eluv-io/ecobra-go` provides packages easing work with [cobra](https://github.com/spf13/cobra):

* `bflags`: a framework for binding command line flags and parameters to fields of structs
* `app`: builds on `bflags` to build complete - or complex - command line applications in a simple and
  declarative fashion.
* `params`: add support to some commonly used parameter types like json and os files including piping from/to stdin
  or stdout

### bflags

Package `bflags` provides binding of command line flags and parameters to fields of structs through annotations tags. 

Tags are specified using 'cmd' followed by either 'flag' or 'arg' and then flags attributes:

```
 flag, name, usage, shorthand, persistent, required, hidden
 arg,  name, usage, order
```

* `usage` : short description of the flag or command line parameter
* `shorthand` : a one letter shorthand
* `persistent`: 'true' to make the flag persistent
* `required`: 'true' to make the flag required
* `hidden`: 'true' to make the flag hidden
* `order` : for command line parameters, an int specifying the order on the command line. If no order is provided the
  order is taken from fields declaration. Note that the order attribute must be specified on all or none of the `arg`
  fields (you may not have some field with the order specified and some other without)

Not all attributes are required:

```
type myInput struct {
	Ip   net.IP `cmd:"flag,ip,node ip,q"`
	Path string `cmd:"arg"`
}
```

The `usage` attribute will be used by the command line help in the description of the flag.

```
type JsonToCsvSpec struct {
	Template          string               `cmd:"arg,template,template mapping columns to path in json,0"`
	Source            *params.PathOrReader `cmd:"arg,source,path to the source file with json,1"`
	Output            *params.PathOrWriter `cmd:"arg,output,path to the output CSV file,2"`
	SheetName         string               `cmd:"flag,sheet-name,name of the sheet when spreadsheet output"`
	WriteColumns      bool                 `cmd:"flag,write-columns,write column names as the first line in csv output"`
	IgnoreEmptyRecord bool                 `cmd:"flag,ignore-empty,do not write empty records"`
}
```

Bindings are supported for:

* all 'native' types of go (int, float and their flavors, bool, string) and pointer to them
* `net.IP`, `time.Duration`
* slices of all the above. They can be comma or space separated on the command line.
* binding to struct or inner structs is also supported, but inner objects have to be initialized pointers (see
  unit-tests)

`bflags` supports binding to custom types through the `Flagger` interface (
see [flags_custom_test.go](bflags/flags_custom_test.go) for a simple example)

See the [bflags doc](bflags/doc.go) for a full description and sample.

### app

With `app` the entire tree of cobra commands is initialized within a struct literal.<br>
The construct looks familiar to cobra users since most fields of the `app.Cmd` struct used to construct the top
level `app.App` have the same name as their counterpart in `cobra.Command`.

```
func initApp() (*app.App, error) {
	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:                "cli",
			Short:              "Sample Client",
			Long:               "A simple command line tool",
			PersistentPreRunE:  app.CobraFn(initializeApp),
			PersistentPostRunE: app.CobraFn(cleanup),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:      "sample test ",
					Short:    "sample <arg>",
					Category: "tools",
					Example:  "sample the_fox ",
					Args:     "ExactArgs(1)",
					RunE:     app.RunFn(execSample),
					Input:    &InputSample{Port: 8080},
				},
			},
		})
	a, err := app.NewApp(spec, nil)
	if err != nil {
		return nil, err
	}
	a.SetCommandStart(cmdStart)
	a.SetCommandEnd(cmdEnd)
	return a, nil
}
```

Input of command are initialized through the `Input` field which will be the second parameter received by the `RunE` function.

The `RunE` must be a function with two parameters:
* the first parameter of type `app.CmdCtx` is mandatory and can be used as a context to convey key/value pairs
* the second parameter is the instance that was defined as `Input`

The output of a run function can be:

* a single value - in which case it has to be of type `error`
* a pair - in which case the second value has to be of type `error`

```
func execSample(ctx *app.CmdCtx, in *InputSample) (*OutputSample, error) {
}
```

The `app.CmdCtx` context can also be initialized prior to the execution of commands and passed to the application

```
func main() {
	a, err := initApp()
	if err != nil {
		exit(err)
	}
	cmdRoot, err := a.Cobra()
	if err != nil {
		exit(err)
	}
	
	// optionally initialize a command context to pass global value to any command
	cmdCtx := app.NewCmdCtx()
	//cmdCtx.Set("some key", someValue)
	bflags.SetCmdCtx(cmdRoot, cmdCtx)

	err = cmdRoot.Execute()
	exit(err)
}
```

See [app_sample.go](app/example/app_sample.go) for a fully running example (the above).
