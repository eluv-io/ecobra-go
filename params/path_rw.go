package params

import (
	"io"

	"github.com/eluv-io/errors-go"
	flag "github.com/spf13/pflag"
)

var _ flag.Value = (*PathOrReader)(nil)
var _ flag.Value = (*PathOrWriter)(nil)

const (
	PathOrReaderType = "$pathOrReader"
	PathOrWriterType = "$pathOrWriter"
)

type Reader interface {
	io.Reader
	io.Closer
}

type Writer interface {
	io.Writer
	io.Closer
}

// PathOrReader implements flag.Value such as it can be used as an attribute
// in another struct and used with bflags bindings.
//
// This allows the same struct to be used for command line parameter as well as
// in programmatic use.
//
// **notes**
// - the path attributes _must_ be initialized otherwise bindings will error
//   In the example below: `tr := &TestReader{Path: &PathOrReader{}}`
// - the attribute MUST be a pointer not a value (*PathOrReader not PathOrReader)
//
// Example
//   type TestReader struct {
//   	Name string        `cmd:"flag"`
//   	Path *PathOrReader `cmd:"flag"`
//   }
type PathOrReader struct {
	Path string
	Read Reader
}

func (p *PathOrReader) Type() string {
	return PathOrReaderType
}

func (p *PathOrReader) String() string {
	if p == nil {
		return ""
	}
	return p.Path
}

func (p *PathOrReader) Set(path string) error {
	if p == nil {
		return errors.E("Set", errors.K.Invalid, "reason", "PathOrReader is nil")
	}
	if p.Read != nil {
		return errors.E("Set", errors.K.Invalid, "reason", "reader is set")
	}
	p.Path = path
	return nil
}

func (p *PathOrReader) CanRead() bool {
	return p != nil && (p.Read != nil || p.Path != "")
}

func (p *PathOrReader) Open() (Reader, error) {
	if p.Read == nil {
		fp := FilePath(p.Path)
		r, err := fp.Open()
		if err != nil {
			return nil, err
		}
		p.Read = r
	}
	return p.Read, nil
}

// PathOrWriter implements flag.Value such as it can be used as an attribute
// in another struct and used with bflags bindings.
//
// This allows the same struct to be used for command line parameter as well as
// in programmatic use.
//
// **notes**
// - a PathOrWriter attribute _must_ be initialized for bindings to work.
// - the attribute MUST be a pointer not a value (*PathOrWriter not PathOrWriter)
//
// see PathOrReader for an example.
type PathOrWriter struct {
	Path  string
	Write Writer
}

func (p *PathOrWriter) Type() string {
	return PathOrWriterType
}

func (p *PathOrWriter) String() string {
	if p == nil {
		return ""
	}
	return p.Path
}

func (p *PathOrWriter) Set(path string) error {
	if p == nil {
		return errors.E("Set", errors.K.Invalid, "reason", "PathOrWriter is nil")
	}
	if p.Write != nil {
		return errors.E("Set", errors.K.Invalid, "reason", "writer is set")
	}
	p.Path = path
	return nil
}

func (p *PathOrWriter) CanWrite() bool {
	return p != nil && (p.Write != nil || p.Path != "")
}

func (p *PathOrWriter) Create() (Writer, error) {
	if p.Write == nil {
		fp := FilePath(p.Path)
		w, err := fp.Create()
		if err != nil {
			return nil, err
		}
		p.Write = w
	}
	return p.Write, nil
}
