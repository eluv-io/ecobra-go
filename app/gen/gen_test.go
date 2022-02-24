package main

import (
	"testing"

	"github.com/eluv-io/ecobra-go/app"
	"github.com/stretchr/testify/require"
)

func TestCmdName(t *testing.T) {
	c := &app.Cmd{
		Use: "content",
	}
	require.Equal(t, "content", cmdName(c, 0))
	require.Equal(t, "Content", cmdName(c, 1))
}
