package params

import (
	"encoding/json"

	"github.com/eluv-io/errors-go"
)

// Json is a string meant to hold a json value.
// - as a string: `{ "bucket": "name", "object": "/path/to/file" }`
// - as a reference to a file: "@dir/file"
// - from piped input: "-"
type Json string

// Map returns the content of the Json string as a map[string]interface
// If the string value starts with '@', it looks for a file with that name and
// uses the content of the file.
func (j Json) Map() (map[string]interface{}, error) {
	e := errors.Template("map", errors.K.IO, "json", j)
	bb, err := BytesFrom(string(j))
	if err != nil {
		return nil, err
	}
	if len(bb) == 0 {
		return nil, nil
	}
	m := map[string]interface{}{}
	err = json.Unmarshal(bb, &m)
	if err != nil {
		return nil, e(err)
	}
	return m, nil
}

func (j Json) Unmarshal(v interface{}) error {
	e := errors.Template("unmarshal", errors.K.IO, "json", j)
	bb, err := BytesFrom(string(j))
	if err != nil {
		return err
	}
	s := string(bb)
	if s == "" {
		return e("reason", "empty string")
	}
	err = json.Unmarshal([]byte(s), v)
	if err != nil {
		return e(err)
	}
	return nil
}
