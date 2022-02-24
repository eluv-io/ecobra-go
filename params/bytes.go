package params

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/eluv-io/errors-go"
)

// Bytes is a string meant to hold a slice of bytes.
// - as a string: `{ "bucket": "name", "object": "/path/to/file" }`
// - as a reference to a file: "@dir/file"
// - from piped input: "-"
type Bytes []byte

func BytesFrom(s string) ([]byte, error) {
	e := errors.Template("from", errors.K.IO, "value", s)

	// read from pipe
	if s == "-" {
		f := FilePath(s)
		r, err := f.Open()
		if err != nil {
			return nil, err
		}
		bb, err := ioutil.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return nil, err
		}
		return bb, nil
	}

	bb := []byte(s)
	// starts-with @: look for file, read it into s
	if strings.Index(s, "@") == 0 {
		f, err := os.Open(s[1:])
		if err != nil {
			return nil, e(err)
		}
		bb, err = ioutil.ReadAll(f)
		_ = f.Close()
		if err != nil {
			return nil, e(err)
		}
	}
	return bb, nil
}
