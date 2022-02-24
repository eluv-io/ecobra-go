package app

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
	"unicode"

	"github.com/spf13/cobra"
)

// templateFuncs is a copy of the cobra.templateFuncs that is maintained in sync
// with the original one: some template are executed from the cobra package while
// other are executed from this package. Both are sharing the same functions.
var templateFuncs = template.FuncMap{
	"trim":                    strings.TrimSpace,
	"trimRightSpace":          trimRightSpace,
	"trimTrailingWhitespaces": trimRightSpace,
	"appendIfNotPresent":      appendIfNotPresent,
	"rpad":                    rpad,
	"gt":                      cobra.Gt,
	"eq":                      cobra.Eq,
}

// rootUsageTemplate is the template used for the root command.
// It adds categories to the default cobra usage template.
var rootUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

These are commands grouped by area{{range categories .}}{{if gt (len .Cmds) 0}}

{{.Title}}{{range .Cmds}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// fullCmdUsageTemplate adds reporting of arguments to the default cobra
// template (returned by *Command.UsageTemplate)
var fullCmdUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if hasArgs . }}

Arguments:
{{arguments . }}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`

// cmdUsageTemplate is a shorter version of the default usage template that only
// shows the usage, arguments and flags. This is the template used when input
// arguments or flags are invalid.
// It is called by the command Usage / UsageString function.
var cmdUsageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if hasArgs . }}

Arguments:
{{arguments . }}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}
`

// cmdHelpTemplate is like the default help template returned by cobra commands
// except it calls 'fullUsageString' for a command rather than the UsageString
// function of the command.
var cmdHelpTemplate = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if or .Runnable .HasSubCommands}}{{fullUsageString . }}{{end}}`

// configureHelp configures help and usage templates for the given command and
// sub-commands.
func configureHelp(cmdRoot *cobra.Command) *cobra.Command {
	// populate 'help' usage template of sub commands before changing root since
	// if not defined the one from the parent is used.
	// use defH := cmdRoot.UsageTemplate() to get the default from cobra
	defH := cmdUsageTemplate
	for _, c := range cmdRoot.Commands() {
		c.SetUsageTemplate(defH)
		c.SetHelpFunc(cmdHelp)
	}
	cmdRoot.SetUsageTemplate(rootUsageTemplate)
	return cmdRoot
}

func fullUsageString(c *cobra.Command) (string, error) {
	buf := bytes.NewBuffer([]byte{})
	err := tmpl(buf, fullCmdUsageTemplate, c)
	if err != nil {
		return "", err
	}
	return string(buf.Bytes()), nil
}

func cmdHelp(c *cobra.Command, args []string) {
	_ = args
	err := tmpl(c.OutOrStdout(), cmdHelpTemplate, c)
	if err != nil {
		c.Println(err)
	}
}

// AddTemplateFunc adds a template function that's available to Usage and Help
// template generation. Also adds the function to the cobra template functions.
func AddTemplateFunc(name string, tmplFunc interface{}) {
	templateFuncs[name] = tmplFunc
	cobra.AddTemplateFunc(name, tmplFunc)
}

func trimRightSpace(s string) string {
	return strings.TrimRightFunc(s, unicode.IsSpace)
}

// appendIfNotPresent will append stringToAppend to the end of s, but only if it's not yet present in s.
func appendIfNotPresent(s, stringToAppend string) string {
	if strings.Contains(s, stringToAppend) {
		return s
	}
	return s + " " + stringToAppend
}

// rpad adds padding to the right of a string.
func rpad(s string, padding int) string {
	templ := fmt.Sprintf("%%-%ds", padding)
	return fmt.Sprintf(templ, s)
}

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}
