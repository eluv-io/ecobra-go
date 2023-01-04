package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/app"
	"github.com/eluv-io/ecobra-go/bflags"
	"github.com/eluv-io/errors-go"
)

type InputSample struct {
	Port     int    `cmd:"flag,port,port to connect to,p"`
	Argument string `cmd:"arg,arg0,command line arg,0"`
}

type OutputSample struct {
	Out string `json:"out"`
}

func execSample(ctx *app.CmdCtx, in *InputSample) (*OutputSample, error) {
	out := &OutputSample{Out: fmt.Sprintf("done with arg: %s, port: %d", in.Argument, in.Port)}
	fmt.Println("executing sample - with context", ctx != nil)
	return out, nil
}

func initializeApp(cmd *cobra.Command, args []string) error {
	e := errors.Template("initializeApp", errors.K.Invalid)
	_ = args

	fmt.Println("initialize")

	m, err := bflags.SetArgs(cmd.Root(), []string{})
	if err != nil {
		return e(err)
	}
	_ = m
	return nil
}

func cleanup(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	fmt.Println("cleanup")
	return nil
}

func cmdStart(c *cobra.Command, flagsAndArgs map[string]string, in interface{}) {
	// from now on errors are execution errors: don't print usage again
	c.SilenceUsage = true
	c.Root().SilenceUsage = true
}

func cmdEnd(c *cobra.Command, out interface{}, err error) {

	if err != nil {
		return
	}
	if out == nil {
		return
	}

	s := ""
	switch res := out.(type) {
	default:
		//json also works for simple types:
		//bool, int, int8, int16, int32, int64, float32, float64, uint, uint8, uint16, uint32, uint64
		buf := bytes.NewBuffer(make([]byte, 0))
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")

		err := enc.Encode(res)
		if err != nil {
			fmt.Println("error formatting output", "command", c.Name(), "error", err)
			return
		}
		s = string(buf.Bytes())
	case *string:
		s = *res
	case string:
		s = res
	}
	fmt.Println(s)
}

func exit(err error) {
	if err == nil {
		return
	}
	fmt.Println(err)
	os.Exit(1)
}

func newApp(appName string) (*app.App, error) {
	spec := app.NewSpec(
		[]*app.CmdCategory{
			{Name: "base", Title: "start working and configure"},
			{Name: "tools", Title: "pre built tools"},
			{Name: "others", Title: "others", Default: true},
		},
		&app.Cmd{
			Use:                appName,
			Short:              "Sample Client",
			Long:               "A simple command line tool",
			PersistentPreRunE:  app.CobraFn(initializeApp),
			PersistentPostRunE: app.CobraFn(cleanup),
			SilenceErrors:      true,
			SubCommands: []*app.Cmd{
				{
					Use:      "sample <arg>",
					Short:    "sample <arg>",
					Category: "tools",
					Example:  "sample the_fox ",
					Args:     "RangeArgs(0,1)",
					RunE:     app.RunFn(execSample),
					Input: &InputSample{
						Argument: "default_arg",
						Port:     8080,
					},
				},
			},
		})
	a, err := app.NewApp(spec, nil)
	if err != nil {
		return nil, err
	}
	return a, nil
}

func initApp(a *app.App) {
	a.SetCommandStart(cmdStart)
	a.SetCommandEnd(cmdEnd)
}

/*
$ go build -o cli ./app/example/
$ ./cli -h
A simple command line tool

Usage:
  cli [command]

These are commands grouped by area

start working and configure
  help        Help about any command

pre built tools
  sample      sample <arg>

Flags:
  -h, --help   help for cli

Use "cli [command] --help" for more information about a command.
*/

/*
$ ./cli sample hello -p 9009
initialize
executing sample - with context true
{
  "out": "done with arg: hello, port: 9009"
}

cleanup
*/
func main() {
	a, err := newApp(os.Args[0])
	if err != nil {
		exit(err)
	}
	initApp(a)

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
