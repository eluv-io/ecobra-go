package bflags

import (
	"bytes"
	"io"
	"text/template"
	_ "unsafe"

	"github.com/spf13/cobra"
)

// templateFuncs exports cobra.templateFuncs
// use the linkname directive to access the private variable
//
//go:linkname templateFuncs github.com/spf13/cobra.templateFuncs
var templateFuncs template.FuncMap

// AddTemplateFunc adds a template function that's available to Usage and Help
// template generation. Also adds the function to the cobra template functions.
func AddTemplateFunc(name string, tmplFunc interface{}) {
	if templateFuncs[name] != nil {
		return
	}
	cobra.AddTemplateFunc(name, tmplFunc)
}

func ConfigureHelpFuncs() {
	AddTemplateFunc("arguments",
		func(cmd *cobra.Command) string {
			argSet, err := GetCmdArgSet(cmd)
			if err != nil {
				return err.Error()
			}
			return argSet.ArgUsages()
		})
	AddTemplateFunc("hasArgs",
		func(cmd *cobra.Command) bool {
			argSet, err := GetCmdArgSet(cmd)
			if err != nil {
				return false
			}
			return len(argSet.Flags) > 0
		})
	AddTemplateFunc("fullUsageString", fullUsageString)
}

func ConfigureCommandHelp(c *cobra.Command) {
	c.SetUsageTemplate(cmdUsageTemplate)
	c.SetHelpFunc(cmdHelp)
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

// tmpl executes the given template text on data, writing the result to w.
func tmpl(w io.Writer, text string, data interface{}) error {
	t := template.New("top")
	t.Funcs(templateFuncs)
	template.Must(t.Parse(text))
	return t.Execute(w, data)
}

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
