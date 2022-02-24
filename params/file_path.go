package params

import (
	"io"
	"os"

	"github.com/eluv-io/errors-go"
	flag "github.com/spf13/pflag"
)

const (
	filePathValueType = "path"
)

// FilePath represents a file path and provides with helper to open / create
// either regular files or pipe stdin / stdout whenever the path is '-'.
type FilePath string

var _ flag.Value = (*FilePath)(nil)

func (f *FilePath) String() string {
	return string(*f)
}

func (f *FilePath) Set(val string) error {
	*f = FilePath(val)
	return nil
}

func (f *FilePath) Type() string {
	return filePathValueType
}

func (f FilePath) IsPipe() bool {
	return f == "-"
}

func (f FilePath) OpenPipe() (Reader, error) {
	e := errors.Template("openPipe", errors.K.IO, "path", f)
	if !f.IsPipe() {
		return nil, e(errors.K.Invalid, "reason", "not a pipe", "path", f)
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, e(err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, e("reason", "no pipe")
	}
	return NopCloser(os.Stdin), nil
}

func (f FilePath) Open() (Reader, error) {
	if f.IsPipe() {
		return f.OpenPipe()
	}
	return os.Open(string(f))
}

func (f FilePath) Create() (Writer, error) {
	e := errors.Template("create", errors.K.IO, "path", f)
	if f == "-" {
		fi, err := os.Stdout.Stat()
		if err != nil {
			return nil, e(err)
		}
		_ = fi
		//if fi.Mode()&os.ModeCharDevice != 0 {
		//	return nil, e("reason", "can't pipe")
		//}
		return NopWriteCloser(os.Stdout), nil
	}
	return os.Create(string(f))
}

func NopCloser(f io.Reader) Reader {
	return &nopCloser{Reader: f}
}

func NopWriteCloser(w io.Writer) Writer {
	return &nopWriteCloser{Writer: w}
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func (w *nopWriteCloser) UnmarshalText(text []byte) error {
	_, err := w.Writer.Write(text)
	return err
}

/* === Pipe notes ===
func main() {
	fi, err := os.Stdin.Stat()
	if err != nil {
		fmt.Println("err:", err)
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		fmt.Println("err:", "no pipe!")
		return
	}
	bb := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(bb)
		if err != nil {
			if err != io.EOF {
				fmt.Println("rerr:", err)
			}
			break
		}
		fmt.Print(string(bb))
		if bb[0] == []byte("e")[0] {
			break
		}
	}
}
*/
