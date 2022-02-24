/*
	Package app makes easier defining - in a structured way - a command line
	application with cobra.Command objects.

	Inputs of commands are bound to flags and args using the bflags package.


	type InputSample struct {
		MyValue string `cmd:"arg" json:"my_value"`
	}

	func execSample(ctx *app.CmdCtx, in *InputSample) (*OutputSample, error) {
		out := &OutputSample{Out: "done"}
		fmt.Println("executing sample", "ctx", ctx != nil, "in", in, "-> out", out)
		return out, nil
	}

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
			Long: "A command line tool",
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

*/
package app
