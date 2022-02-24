package main

import (
	"fmt"
	"os"

	"github.com/eluv-io/ecobra-go/app"
)

func Execute() {
	a, err := newApp("")
	root, err := a.Cobra()
	if err != nil {
		exit(err)
	}
	initApp(a)

	os.Args = []string{"cli", "sample", "my arg"}
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}

	a, _ = app.NewApp(a.Spec(), nil)
	initApp(a)
	root, _ = a.Cobra()
	os.Args = []string{"cli", "sample", "my_fox", "-p", "9"}
	err = root.Execute()
	if err != nil {
		fmt.Println("error", err)
		return
	}
}

func ExampleExecute() {

	Execute()

	// Output:
	// initialize
	// executing sample - with context true
	// {
	//   "out": "done with arg: my arg, port: 8080"
	// }
	//
	// cleanup
	// initialize
	// executing sample - with context true
	// {
	//   "out": "done with arg: my_fox, port: 9"
	// }
	//
	// cleanup
}
