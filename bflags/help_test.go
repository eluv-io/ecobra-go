package bflags

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestExpectedTemplateFuncs makes sure templateFuncs is synced with cobra.templateFuncs
func TestExpectedTemplateFuncs(t *testing.T) {
	for _, name := range []string{
		"trim",
		"trimRightSpace",
		"trimTrailingWhitespaces",
		"appendIfNotPresent",
		"rpad",
		"gt",
		"eq",
	} {
		require.NotNil(t, templateFuncs[name])
	}
	// works if the test is run alone, but len is 10 if the singleton was already updated
	//require.Equal(t, 7, len(templateFuncs))
	ConfigureHelpFuncs()
	require.Equal(t, 10, len(templateFuncs))
	for _, name := range []string{
		"arguments",
		"hasArgs",
		"fullUsageString",
	} {
		require.NotNil(t, templateFuncs[name])
	}
}
