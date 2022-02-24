package app_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/eluv-io/ecobra-go/app"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

var appSample = `{
	"categories": [
		{"name": "base",       "title": "start working and configure"},
		{"name": "tools",      "title": "pre built tools" },
		{"name": "others",     "title": "others", "default": true }
	],
	"cmd_root": {
		"use": "cli ",
		"short": "Sample Client",
		"long": ["A command line tool",
				"that uses REST API"],
		"persistent_pre_run_e": "initializeSample",
		"persistent_post_run_e": "cleanup",
		"silence_errors": true,
		"sub_commands": [{
			"use": "sample test ",
			"short": "Sample <arg>",
			"long": ["sample my_arg"],
			"category": "tools",
			"example": ["sample fox"],
			"run_e": "execSample",
			"input_ctor": "sample",
			"input": {
				"my_value": "xyz"
			}
		}]
	}
}`

// ---------------------------------- SETUP / TEAR-DOWN func

func initializeSample(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	fmt.Println("initializeSample")
	return nil
}
func cleanup(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	fmt.Println("cleanup")
	return nil
}

// ---------------------------------- LEGACY
// legacy initialisation
func initCmd() (*cobra.Command, error) {
	var cmdRoot = &cobra.Command{
		Use:   "cli",
		Short: "Sample Client",
		Long: `
A command line tool that uses xx REST API,
Reference: http://example.com`,
		PersistentPreRunE:  initializeSample,
		PersistentPostRunE: cleanup,
		SilenceErrors:      true,
	}

	var cmdSample = &cobra.Command{
		Use:     "sample",
		Short:   "sample <arg>",
		Long:    `sample <fox>`,
		Example: "sample the_fox",
		RunE:    runSample,
	}
	cmdRoot.AddCommand(cmdSample)
	return cmdRoot, nil
}

// legacy function
func runSample(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	fmt.Println("run sample")
	// here deal with flags and params
	in := &InputSample{}
	ctx := &app.CmdCtx{}
	out, err := execSample(ctx, in)
	if err != nil {
		return err
	}
	fmt.Println("out", out)
	return nil
}

// ---------------------------------- EOF LEGACY

// for sample
type InputSample struct {
	MyValue string `cmd:"arg" json:"my_value"`
}

type OutputSample struct {
	Out string
}

func execSample(ctx *app.CmdCtx, in *InputSample) (*OutputSample, error) {
	out := &OutputSample{Out: "done"}
	fmt.Println("executing sample", "ctx", ctx != nil, "in", in, "-> out", out)
	return out, nil
}

var cobraFns = map[string]app.CobraFunction{
	"initializeSample": initializeSample,
	"cleanup":          cleanup,
}
var runFns = map[string]app.Runfn{
	"execSample": execSample,
}
var inputs = map[string]app.Ctor{
	"sample": func() interface{} { return &InputSample{} },
}

func TestCommandSerial(t *testing.T) {
	_, err := initCmd()
	require.NoError(t, err)

	rt, err := app.RtFunctions(cobraFns, inputs, runFns)
	require.NoError(t, err)
	a, err := app.NewAppFromSpec(appSample, rt)
	require.NoError(t, err)

	bb, err := json.MarshalIndent(a.Spec(), "", "  ")
	require.NoError(t, err)
	fmt.Println(string(bb))
}

func ExampleLegacy() {
	root, err := initCmd()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	os.Args = append([]string{""}, "sample")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	// Output:
	// initializeSample
	// run sample
	// executing sample ctx true in &{} -> out &{done}
	// out &{done}
	// cleanup

}

// ExampleCommandFromJson shows an app example built from a json definition: RT
// functions are defined by their name in the json spec.
// - see ExampleCommandWithStrings for example wit spec defined inline & RT
//   function still defined by name
// - see ExampleCommand for example with all RT function defined inline
func ExampleCommandFromJson() {
	rt, err := app.RtFunctions(cobraFns, inputs, runFns)
	if err != nil {
		fmt.Println("error", err)
		return
	}
	a, err := app.NewAppFromSpec(appSample, rt)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	root, err := a.Cobra()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	os.Args = append([]string{""}, "sample")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	// Output:
	// initializeSample
	// executing sample ctx true in &{xyz} -> out &{done}
	// cleanup

}

// same as ExampleCommandFromJson but the spec is defined inline in code
// RT functions are still defined by name
// see ExampleCommand for example with all RT function defined inline
func ExampleCommandWithStrings() {
	rt, err := app.RtFunctions(cobraFns, inputs, runFns)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:   "cli ",
			Short: "Sample Client",
			Long: "A command line tool" +
				"that uses xx REST API",
			PersistentPreRunE:  app.CobraFnWithName("initializeSample"),
			PersistentPostRunE: app.CobraFnWithName("cleanup"),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:       "sample test ",
					Short:     "sample <arg>",
					Category:  "tools",
					Example:   "sample the_fox ",
					RunE:      app.RunFnWithName("execSample"),
					InputCtor: "sample",
					Input:     map[string]interface{}{"my_value": "xyz"},
				},
			},
		})
	a, err := app.NewApp(spec, rt)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	root, err := a.Cobra()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	os.Args = append([]string{""}, "sample")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	// Output:
	// initializeSample
	// executing sample ctx true in &{xyz} -> out &{done}
	// cleanup
}

// same as ExampleCommandWithStrings but no RT functions are used: they're all
// defined inline in code.
func ExampleCommand() {
	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:   "cli ",
			Short: "Sample Client",
			Long: "A command line tool\n" +
				"that uses xx REST API",
			PersistentPreRunE:  app.CobraFn(initializeSample),
			PersistentPostRunE: app.CobraFn(cleanup),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:      "sample test ",
					Short:    "sample <arg>",
					Category: "tools",
					Example:  "smaple the_fox ",
					RunE:     app.RunFn(execSample),
					Input:    &InputSample{MyValue: "xyz"},
				},
			},
		})
	a, err := app.NewApp(spec, nil)
	if err != nil {
		fmt.Println("error", err)
		return
	}

	root, err := a.Cobra()
	if err != nil {
		fmt.Println("error", err)
		return
	}
	os.Args = append([]string{""}, "sample")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}

	a, _ = app.NewApp(spec, nil)
	root, _ = a.Cobra()
	os.Args = append([]string{""}, "sample", "my_fox")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}

	// Output:
	// initializeSample
	// executing sample ctx true in &{xyz} -> out &{done}
	// cleanup
	// initializeSample
	// executing sample ctx true in &{my_fox} -> out &{done}
	// cleanup

}

func execSample2(ctx *app.CmdCtx, in *InputSample) error {
	out := &OutputSample{Out: "done"}
	fmt.Println("executing sample", "ctx", ctx != nil, "in", in, "-> out", out)
	if in.MyValue == "error" {
		return fmt.Errorf("raising test error")
	}
	return nil
}

func doneCmd(cmd *cobra.Command, out interface{}, err error) {
	fmt.Println("done", cmd.Name(), "out", out, "err", err)
}

func TestCommandOneReturnedValue(t *testing.T) {
	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:   "cli ",
			Short: "Sample Client",
			Long: "A command line tool\n" +
				"that uses xx REST API",
			PersistentPreRunE:  app.CobraFn(initializeSample),
			PersistentPostRunE: app.CobraFn(cleanup),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:      "sample test ",
					Short:    "sample <arg>",
					Category: "tools",
					Example:  "smaple the_fox ",
					RunE:     app.RunFn(execSample2),
					Input:    &InputSample{MyValue: "xyz"},
				},
			},
		})
	a, err := app.NewApp(spec, nil)
	if err != nil {
		fmt.Println("error", err)
		return
	}
	a.SetCommandEnd(doneCmd)

	root, err := a.Cobra()
	if err != nil {
		fmt.Println("got error", err)
		return
	}
	os.Args = append([]string{""}, "sample", "my_fox")
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}

	a, _ = app.NewApp(spec, nil)
	root, err = a.Cobra()
	if err != nil {
		fmt.Println("got error", err)
		return
	}
	os.Args = append([]string{""}, "sample", "error")
	err = root.Execute()
	require.Error(t, err)

	// Output:
	// initializeSample
	// executing sample ctx true in &{my_fox} -> out &{done}
	// done sample out <nil> err <nil>
	// cleanup
	// .. error on second call

}

func TestMarshallSpec(t *testing.T) {
	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:   "cli ",
			Short: "Sample Client",
			Long: "A command line tool\n" +
				"that uses xx REST API",
			PersistentPreRunE:  app.CobraFn(initializeSample),
			PersistentPostRunE: app.CobraFn(cleanup),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:       "sample test ",
					Short:     "sample <arg>",
					Category:  "tools",
					Example:   "smaple the_fox ",
					Args:      "NoArgs",
					RunE:      app.RunFn(execSample),
					InputCtor: "",
					Input:     &InputSample{MyValue: "xyz"},
					//InputCtor: "sample",
					//Input:     app.CmdInput{"my_value": "xyz"},
				},
			},
		})
	a, err := app.NewApp(spec, nil)
	require.NoError(t, err)

	root, err := a.Cobra()
	require.NoError(t, err)
	os.Args = append([]string{""}, "sample")
	err = root.Execute()
	require.NoError(t, err)

	s, err := json.MarshalIndent(spec, "", "  ")
	require.NoError(t, err)
	fmt.Println(string(s))
}
