/*
	Gen provides a rudimentary code generator for app.
	It reads a json definition of a hierarchy of commands and outputs a go file
	containing generated
	- commands with their input and output structs
	- the command hierarchy in go code enriched with the above.
	The go package is always 'cmd'

	The following rules apply:
	- commands that have sub-commands do not have input / output structs and run
      function generated
    - an input struct or run function that is specified	in the json spec is not
	  generated as the generator assumes it already exists.
	- a 'base' name is generated for each command by concatenating names in the
   	  json path (for example the command add / stuff generates a base name addStuff)
	  - inputs are named baseIn
	  - outputs are named baseOut
	  - run functions are named runBase
	- some fields are ignored:
	  - pre and post functions
	  - Annotations
	  - Aliases
	  - SuggestFor
	  - ValidArgs
	  - ArgAliases
	  - SuggestionsMinimumDistance

	The command accepts a single mandatory argument: the path to the json spec
	and always outputs its result to stdout.
*/
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"github.com/eluv-io/errors-go"

	"github.com/eluv-io/ecobra-go/app"
)

const pkg = "cmd"

// sol: start of line
func sol(tabs int, sb *strings.Builder) {
	sb.WriteString(strings.Repeat("\t", tabs))
}

// eol: end of line
func eol(sb *strings.Builder) {
	sb.WriteString("\n")
}

// nl: new line starting with 'tabs' tabs followed by val and a line feed
func nl(tabs int, val string, sb *strings.Builder) {
	sol(tabs, sb)
	sb.WriteString(val)
	eol(sb)
}

func cmdName(c *app.Cmd, level int) string {
	n := c.Name()
	ndx := strings.Index(n, "-")
	for ndx >= 0 {
		c := ""
		if ndx < len(n)-1 {
			c = strings.ToUpper(n[ndx+1:ndx+2]) + n[ndx+2:]
		}
		n = n[:ndx] + c
		ndx = strings.Index(n, "-")
	}

	if level == 0 {
		return strings.ToLower(n[0:1]) + n[1:]
	}
	return strings.ToUpper(n[0:1]) + n[1:]
}

func cmdRunName(c *app.Cmd, path []string) string {
	n := cmdBaseName(c, path, 1)
	r := strings.ToUpper(n[:1])
	return "run" + r + n[1:]
}

func cmdBaseName(c *app.Cmd, path []string, level int) string {
	return strings.Join(path, "") + cmdName(c, level)
}

func cmdInputName(c *app.Cmd, path []string, level int) string {
	return cmdBaseName(c, path, level) + "In"
}

func cmdOutputName(c *app.Cmd, path []string, level int) string {
	return cmdBaseName(c, path, level) + "Out"
}

func writeFuncDecl(level int, name, value string, sb *strings.Builder) {
	nl(2*(level+1), name+": app.RunFn("+value+"),", sb)
}

func writeStringDecl(level int, name, value string, sb *strings.Builder) {
	if value == "" {
		return
	}
	nl(2*(level+1), name+": `"+value+"`,", sb)
}

func writeInputDecl(level int, name, value string, sb *strings.Builder) {
	if value == "" {
		return
	}
	nl(2*(level+1), name+": "+value+",", sb)
}

func writeBoolDecl(level int, name string, value bool, sb *strings.Builder) {
	if !value {
		return
	}
	nl(2*(level+1), name+": \""+strconv.FormatBool(value)+"\",", sb)
}

func genCmdFunc(c *app.Cmd, path []string, level int, funs *strings.Builder) {

	nl(0, "", funs)
	bn := "//----- " + cmdBaseName(c, path, level)
	nl(0, bn+strings.Repeat("-", 80-len(bn)), funs)

	inStruct := cmdInputName(c, path, level)
	outStruct := cmdOutputName(c, path, level)
	t := 0
	if c.Input == nil {
		nl(t, "type "+inStruct+" struct {", funs)
		nl(t, "}", funs)
		nl(t, "", funs)
	}
	if c.RunE.IsNil() {
		nl(t, "type "+outStruct+" struct {", funs)
		nl(t, "}", funs)
		nl(t, "", funs)

		runName := cmdRunName(c, path)
		nl(t, "func "+runName+"(c *app.CmdCtx, _ *"+inStruct+") (*"+outStruct+", error) {", funs)
		nl(t+1, "ctx := ctx(c)", funs)
		nl(t+1, "_ = ctx", funs)
		nl(t+1, "return nil, nil", funs)
		nl(t, "}", funs)
	}
}

func genCmdStartDecl(c *app.Cmd, level int, decl *strings.Builder) {
	s := "{"
	if level == 0 {
		nl(2*level, "func init"+cmdName(c, 1)+"() *app.Cmd {", decl)
		s = "return &app.Cmd{"
	}
	nl(1+2*level, s, decl)
}

func genCmdDecl(c *app.Cmd, path []string, level int, decl *strings.Builder) (string, error) {
	n := cmdName(c, level)
	writeStringDecl(level, "Use", c.Use, decl)
	writeStringDecl(level, "Short", c.Short, decl)
	writeStringDecl(level, "Long", string(c.Long), decl)
	writeStringDecl(level, "Category", c.Category, decl)
	writeStringDecl(level, "Example", string(c.Example), decl)
	writeStringDecl(level, "Args", c.Args, decl)
	writeStringDecl(level, "BashCompletionFunction", c.BashCompletionFunction, decl)
	writeStringDecl(level, "Deprecated", c.Deprecated, decl)
	writeBoolDecl(level, "Hidden", c.Hidden, decl)
	writeStringDecl(level, "Version", c.Version, decl)
	if c.RunE.IsNil() && len(c.SubCommands) == 0 {
		writeFuncDecl(level, "RunE", cmdRunName(c, path), decl)
	}
	writeBoolDecl(level, "SilenceErrors", c.SilenceErrors, decl)
	writeBoolDecl(level, "SilenceUsage", c.SilenceUsage, decl)
	writeBoolDecl(level, "DisableFlagParsing", c.DisableFlagParsing, decl)
	writeBoolDecl(level, "DisableAutoGenTag", c.DisableAutoGenTag, decl)
	writeBoolDecl(level, "DisableFlagsInUseLine", c.DisableFlagsInUseLine, decl)
	writeBoolDecl(level, "DisableSuggestions", c.DisableSuggestions, decl)
	writeBoolDecl(level, "TraverseChildren", c.TraverseChildren, decl)
	if c.Input == nil && len(c.SubCommands) == 0 {
		writeInputDecl(level, "Input", "&"+cmdInputName(c, path, level)+"{}", decl)
	}
	return n, nil
}

func genCmdEndDecl(level int, decl *strings.Builder) {
	s := "}"
	if level > 0 {
		s += ","
	}
	nl(1+2*level, s, decl)
	if level == 0 {
		nl(2*level, "}", decl)
	}
}

func genCmdStartSubs(level int, decl *strings.Builder) {
	nl(2+2*level, "SubCommands: []*app.Cmd{", decl)
}

func genCmdEndSubs(level int, decl *strings.Builder) {
	nl(2+2*level, "},", decl)
}

func visit(c *app.Cmd, path []string, level int, decl, funs *strings.Builder) error {
	if c.Use == "" {
		return errors.E("Empty cmd use", errors.K.Invalid, "path", strings.Join(path, ","))
	}
	if len(c.SubCommands) == 0 {
		genCmdFunc(c, path, level, funs)
	}
	genCmdStartDecl(c, level, decl)
	name, err := genCmdDecl(c, path, level, decl)
	if err != nil {
		return err
	}
	if len(c.SubCommands) > 0 {
		genCmdStartSubs(level, decl)
		for _, sub := range c.SubCommands {
			err = visit(sub, append(path, name), level+1, decl, funs)
			if err != nil {
				return err
			}
		}
		genCmdEndSubs(level, decl)
	}
	genCmdEndDecl(level, decl)
	return nil
}

func genHeader(decl *strings.Builder) {
	decl.WriteString("package " + pkg + "\n\n")
	decl.WriteString("import " + "(\n")
	decl.WriteString("\t" + `"eluvio/client/ecobra/app"` + "\n")
	decl.WriteString(")\n\n")
}

func generate(a *app.App) (string, error) {
	c := a.Spec().CmdRoot
	decl := &strings.Builder{}
	funs := &strings.Builder{}
	genHeader(decl)
	err := visit(c, []string{}, 0, decl, funs)
	if err != nil {
		return "", err
	}
	return strings.Join([]string{decl.String(), funs.String()}, "\n"), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("gen <path to json specification of app>")
		os.Exit(1)
	}
	sp, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Error reading json", os.Args[1], err)
		os.Exit(1)
	}

	a, err := app.NewAppFromSpec(string(sp), nil)
	if err != nil {
		fmt.Println("Error parsing app spec", os.Args[1], err)
		os.Exit(1)
	}

	goc, err := generate(a)
	if err != nil {
		fmt.Println("Error generating app", os.Args[1], err)
		os.Exit(1)
	}
	fmt.Println(goc)
}
