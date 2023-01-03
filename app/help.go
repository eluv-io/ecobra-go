package app

import (
	"github.com/spf13/cobra"

	"github.com/eluv-io/ecobra-go/bflags"
)

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

// AddTemplateFunc adds a template function that's available to Usage and Help
// template generation. Also adds the function to the cobra template functions.
func AddTemplateFunc(name string, tmplFunc interface{}) {
	bflags.AddTemplateFunc(name, tmplFunc)
}

// configureHelp configures help and usage templates for the given command and
// sub-commands.
func configureHelp(cmdRoot *cobra.Command) *cobra.Command {
	// populate 'help' usage template of sub commands before changing root since
	// if not defined the one from the parent is used.
	// use defH := cmdRoot.UsageTemplate() to get the default from cobra
	for _, c := range cmdRoot.Commands() {
		bflags.ConfigureCommandHelp(c)
	}
	cmdRoot.SetUsageTemplate(rootUsageTemplate)
	return cmdRoot
}
