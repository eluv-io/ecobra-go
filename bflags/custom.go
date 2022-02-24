package bflags

import (
	"reflect"

	flag "github.com/spf13/pflag"
)

type Flagged struct {
	Ptr      interface{} // a pointer to the original value (or the original value itself)
	Flag     flag.Value  // a flag.Value representing the value
	CsvSlice bool        // true if the value is a slice whose string representation is comma separated
}

type Flagger interface {

	// Bind returns true if this Flagger handles the given type
	Bind(t reflect.Type) bool

	// Flag returns a Flagged or nil if not handled.
	Flag(val interface{}) *Flagged
}
