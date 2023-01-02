package bflags

import (
	"fmt"
	"net"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
)

const (
	bflagsLogPath = "/cli/bflags"
)

var log = elog.Get(bflagsLogPath)

type bindOpts struct {
}

// An flagsBinder binds flags and args into a *cobra.Command.
type flagsBinder struct {
	cmd      *cobra.Command
	custom   Flagger
	cmdFlags CmdFlags
	argFlags []*FlagBond
}

func (e *flagsBinder) Reset(c *cobra.Command, custom Flagger) {
	e.cmd = c
	e.custom = custom
	e.cmdFlags = make(CmdFlags)
	e.argFlags = make([]*FlagBond, 0)
}

type bindError struct{ error }

var bindStatePool sync.Pool

func newFlagsBinder(c *cobra.Command, custom Flagger) *flagsBinder {
	var e *flagsBinder
	if v := bindStatePool.Get(); v != nil {
		e = v.(*flagsBinder)
	} else {
		e = new(flagsBinder)
	}
	e.Reset(c, custom)
	return e
}

func (e *flagsBinder) bind(v interface{}, opts bindOpts) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if je, ok := r.(bindError); ok {
				err = je.error
			} else {
				panic(r)
			}
		}
	}()
	ex := errors.Template("bind", "v", fmt.Sprintf("%#v", v))
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Interface && val.Kind() != reflect.Ptr {
		return ex("reason", "cannot call value.Elem",
			"kind", val.Kind().String())
	}
	e.reflectValue(val.Elem(), opts)
	err = e.cmdFlags.ConfigureCmd(e.cmd, e.custom)
	if err != nil {
		return err
	}
	// configure args and deal with ordering
	argf := make([]*FlagBond, len(e.argFlags))
	if len(e.argFlags) > 0 {
		hasOrder := e.argFlags[0].ArgOrder >= 0
		for i := 1; i < len(e.argFlags); i++ {
			if (e.argFlags[i].ArgOrder >= 0) != hasOrder {
				return ex("reason", "args order must be either fully specified or not at all",
					"arg_flag", e.argFlags[i].Name,
					"order", e.argFlags[i].ArgOrder)
			}
		}
	}
	for i, fb := range e.argFlags {
		fb.Hidden = true
		_, err = e.cmdFlags.configureFlag(e.cmd, e.custom, fb)
		if err != nil {
			return err
		}
		if fb.ArgOrder >= 0 {
			if fb.ArgOrder >= len(e.argFlags) {
				return ex("reason", "invalid order specified", "found order", fb.ArgOrder)
			}
			if argf[fb.ArgOrder] != nil {
				return ex("reason", "duplicate order specified", "found order", fb.ArgOrder)
			}
			argf[fb.ArgOrder] = fb
		} else {
			argf[i] = fb
		}
	}
	setCmdArgSet(e.cmd, argf)
	setCmdInput(e.cmd, v)

	return nil
}

// error aborts the binding by panicking with err wrapped in bindError.
func (e *flagsBinder) error(err *errors.Error) {
	panic(bindError{error: err})
}

func (e *flagsBinder) reflectValue(v reflect.Value, opts bindOpts) {
	e.valueBinder(v)(e, v, nil, opts)
}

func (e *flagsBinder) setFlagBound(ptr interface{}, spec cmdSpec) {
	ex := errors.Template("setFlagBound",
		"tag_type", spec.kind(),
		"name", spec.getName())

	short := ""
	required := false
	persistent := false
	hidden := false
	order := -1
	isArg := false
	if spec.kind() == flagTag {
		short = spec.(*flagSpec).shorthand
		persistent = spec.(*flagSpec).persistent
		required = spec.(*flagSpec).required
		hidden = spec.(*flagSpec).hidden
	} else {
		isArg = true
		order = spec.(*argSpec).order
	}

	name := cmdFlag(spec.getName())
	fb := &FlagBond{
		isArg:       isArg,
		Name:        name,
		Shorthand:   short,
		Value:       ptr,
		Usage:       spec.getDescription(),
		Required:    required,
		Persistent:  persistent,
		Hidden:      hidden,
		ArgOrder:    order,
		Annotations: spec.getAnnotations(),
	}

	if spec.kind() == flagTag {
		if _, ok := e.cmdFlags[name]; ok {
			e.error(ex("reason", "duplicate flag", "name", name))
		}
		e.cmdFlags[name] = fb
	} else {
		e.argFlags = append(e.argFlags, fb)
	}
}

func (e *flagsBinder) valueBinder(v reflect.Value) binderFunc {
	if !v.IsValid() {
		return invalidValueBinder
	}
	return typeBinder(e, v.Type())
}

// ----- binders -----

type binderFunc func(e *flagsBinder, v reflect.Value, spec cmdSpec, opts bindOpts)

func invalidValueBinder(e *flagsBinder, v reflect.Value, _ cmdSpec, _ bindOpts) {
	e.error(errors.E("invalid value", "value", v))
}

func unsupportedTypeBinder(e *flagsBinder, v reflect.Value, _ cmdSpec, _ bindOpts) {
	e.error(errors.E("unsupported type", "type", v.Type()))
}

func interfaceBinder(e *flagsBinder, v reflect.Value, _ cmdSpec, opts bindOpts) {
	if v.IsNil() {
		e.error(errors.E("unsupported type", "type", v.Type(), "reason", "nil interface"))
		return
	}
	e.reflectValue(v.Elem(), opts)
}

func newMapBinder(_ reflect.Type) binderFunc {
	return unsupportedTypeBinder
}

// sliceBinder just wraps an arrayBinder
type sliceBinder struct {
	arrayEnc binderFunc
}

func (se *sliceBinder) bind(e *flagsBinder, v reflect.Value, spec cmdSpec, opts bindOpts) {
	se.arrayEnc(e, v, spec, opts)
}

func newSliceBinder(e *flagsBinder, t reflect.Type) binderFunc {
	enc := &sliceBinder{newArrayBinder(e, t)}
	return enc.bind
}

type arrayBinder struct {
	elemTyp reflect.Type
}

func (ae *arrayBinder) bind(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("arrayBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	ok := v.Kind() == reflect.Slice || v.Kind() == reflect.Array
	if !ok {
		e.error(ex("wrong type, expected slice or array, got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(iface, spec)
}

func newArrayBinder(e *flagsBinder, t reflect.Type) binderFunc {
	_ = e
	enc := &arrayBinder{t.Elem()}
	return enc.bind
}

type ptrBinder struct {
	elemEnc binderFunc
}

func (pe ptrBinder) bind(e *flagsBinder, v reflect.Value, spec cmdSpec, opts bindOpts) {
	ex := errors.Template("ptrBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())
	if v.IsNil() {
		iface := v.Addr().Interface()
		switch ptr := iface.(type) {
		case **bool, **string, **int, **int64:
			// allow binding to nil of these
			e.setFlagBound(ptr, spec)
		default:
			// others are not supported
			e.error(ex("reason", "can't bind to nil pointer"))
		}
		return
	}
	pe.elemEnc(e, v.Elem(), spec, opts)
}

func newPtrBinder(e *flagsBinder, t reflect.Type) binderFunc {
	enc := ptrBinder{typeBinder(e, t.Elem())}
	return enc.bind
}

func boolBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("boolBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	ptr, ok := iface.(*bool)
	if !ok {
		e.error(ex("wrong type, expected *bool, got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(ptr, spec)
}

func stringBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("stringBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	ptr, ok := iface.(*string)
	if !ok {
		e.error(ex("wrong type, expected *string, got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(ptr, spec)
}

func uintBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("uintBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()

	var ok = false
	k := v.Kind()
	switch k {
	case reflect.Uint:
		fallthrough
	case reflect.Uint8:
		fallthrough
	case reflect.Uint16:
		fallthrough
	case reflect.Uint32:
		fallthrough
	case reflect.Uint64:
		ok = true
	case reflect.Uintptr:
		// needed ?
	}

	if !ok {
		e.error(ex("wrong type, expected *uint[x], got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(iface, spec)
}

func intBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("intBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()

	var ok = false
	k := v.Kind()
	switch k {
	case reflect.Int:
		fallthrough
	case reflect.Int8:
		fallthrough
	case reflect.Int16:
		fallthrough
	case reflect.Int32:
		fallthrough
	case reflect.Int64:
		ok = true
	}

	if !ok {
		e.error(ex("wrong type, expected *int[x], got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(iface, spec)
}

type floatBinder int // number of bits

func (bits floatBinder) bind(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	f := v.Float()
	_ = f
	ex := errors.Template("floatBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()

	var ok = false
	k := v.Kind()
	switch k {
	case reflect.Float32:
		fallthrough
	case reflect.Float64:
		ok = true
	}

	if !ok {
		e.error(ex("wrong type, expected *float[x], got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(iface, spec)
}

var (
	float32Binder = (floatBinder(32)).bind
	float64Binder = (floatBinder(64)).bind
)

func ipBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("ipBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	ptr, ok := iface.(*net.IP)
	if !ok {
		e.error(ex("wrong type, expected *net.IP, got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(ptr, spec)
}

func durationBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("durationBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	ptr, ok := iface.(*time.Duration)
	if !ok {
		e.error(ex("wrong type, expected *time.Duration, got", reflect.TypeOf(iface)))
	}
	e.setFlagBound(ptr, spec)
}

func customBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	iface := v.Addr().Interface()
	e.setFlagBound(iface, spec)
}

func flagValueBinder(e *flagsBinder, v reflect.Value, spec cmdSpec, _ bindOpts) {
	ex := errors.Template("flagValueBinder",
		"tag_type", spec.kind(),
		"name", spec.getName())

	iface := v.Addr().Interface()
	_, ok := reflect.ValueOf(iface).Elem().Interface().(flag.Value)
	if !ok {
		e.error(ex("wrong type, expected flag.Value, got", reflect.TypeOf(iface)))
	}
	if v.Kind() == reflect.Struct {
		//ignored for now: need to use a pointer on it
	}
	e.setFlagBound(iface, spec)
}

type structBinder struct {
	fields    []field
	fieldEncs []binderFunc
}

func (se *structBinder) bind(e *flagsBinder, v reflect.Value, _ cmdSpec, opts bindOpts) {
	for i, f := range se.fields {
		fv := fieldByIndex(v, f.index)
		//if !fv.IsValid() || f.omitEmpty && isEmptyValue(fv) {
		if !fv.IsValid() {
			// NOTE: this where binding fields of inner structs WON'T work if
			//       the inner struct is not initialized (binding won't happen)
			//continue: raise an error rather than ignore
			e.error(errors.E("bind", errors.K.Invalid,
				"reason", "invalid value for binding",
				"possible cause", "inner struct not initialized",
				"name", f.name,
				"type", f.typ.String()))
		}
		se.fieldEncs[i](e, fv, f.spec, opts)
	}
}

func newStructBinder(e *flagsBinder, t reflect.Type) binderFunc {
	fields := typeFields(t)
	se := &structBinder{
		fields:    fields,
		fieldEncs: make([]binderFunc, len(fields)),
	}
	for i, f := range fields {
		se.fieldEncs[i] = typeBinder(e, typeByIndex(t, f.index))
	}
	return se.bind
}

func typeBinder(e *flagsBinder, t reflect.Type) binderFunc {
	// first take care of custom types
	if e.custom != nil && e.custom.Bind(t) {
		return customBinder
	}
	// .. or known types
	switch t {
	case reflect.TypeOf(net.IP{}):
		return ipBinder
	case reflect.TypeOf(time.Duration(0)):
		return durationBinder
	}
	//if t.Implements(reflect.TypeOf((*flag.Value)(nil)).Elem()) {
	//	return flagValueBinder
	//}

	switch t.Kind() {
	case reflect.String:
		return stringBinder
	case reflect.Bool:
		return boolBinder
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intBinder
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintBinder
	case reflect.Float32:
		return float32Binder
	case reflect.Float64:
		return float64Binder
	case reflect.Interface:
		return interfaceBinder
	case reflect.Struct:
		return newStructBinder(e, t)
	case reflect.Map:
		return newMapBinder(t)
	case reflect.Slice:
		return newSliceBinder(e, t)
	case reflect.Array:
		return newArrayBinder(e, t)
	case reflect.Ptr:
		if t.Implements(reflect.TypeOf((*flag.Value)(nil)).Elem()) {
			return flagValueBinder
		}
		return newPtrBinder(e, t)
	default:
		return unsupportedTypeBinder
	}

}

// -----

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for _, i := range index {
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return reflect.Value{}
			}
			v = v.Elem()
		}
		v = v.Field(i)
	}
	return v
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, i := range index {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		t = t.Field(i).Type
	}
	return t
}

/*
	tag spec

cmd:"arg,id,content id,0"
cmd:"flag,id,content id, i,true,true,true"

meta:"val1,val2"
*/
const (
	cmdTag  = "cmd"
	argTag  = "arg"
	flagTag = "flag"
	metaTag = "meta"
)

type cmdSpec interface {
	kind() string
	getName() string
	setName(s string)
	getDescription() string
	getAnnotations() []string
}

// cmd:"arg,[name, description, [order]]"
type argSpec struct {
	name        string   // name of the flag or arg parameter
	description string   // description
	order       int      // optional order on command line
	annotations []string // annotations
}

func (a *argSpec) kind() string {
	return argTag
}

func (a *argSpec) getName() string {
	return a.name
}
func (a *argSpec) setName(s string) {
	a.name = s
}
func (a *argSpec) getDescription() string {
	return a.description
}
func (a *argSpec) getAnnotations() []string {
	return a.annotations
}

// cmd:"flag,name[, description, short hand, persistent=false, required=false, hidden=false]" meta:"val1,val2,val3"
type flagSpec struct {
	name        string   // name of the flag or arg parameter
	description string   // description: used for usage
	shorthand   string   // one letter shorthand or the empty string for none
	persistent  bool     // true: the flag is available to the command as well as every command under the command
	required    bool     // true if the flag is required
	hidden      bool     // true if the flag is hidden
	annotations []string // annotations
}

func (a *flagSpec) kind() string {
	return flagTag
}

func (a *flagSpec) getName() string {
	return a.name
}
func (a *flagSpec) setName(s string) {
	a.name = s
}
func (a *flagSpec) getDescription() string {
	return a.description
}
func (a *flagSpec) getAnnotations() []string {
	return a.annotations
}

// A field represents a single field found in a struct.
type field struct {
	name  string
	index []int
	typ   reflect.Type
	spec  cmdSpec
}

// byIndex sorts field by index sequence.
type byIndex []field

func (x byIndex) Len() int { return len(x) }

func (x byIndex) Swap(i, j int) { x[i], x[j] = x[j], x[i] }

func (x byIndex) Less(i, j int) bool {
	for k, xik := range x[i].index {
		if k >= len(x[j].index) {
			return false
		}
		if xik != x[j].index[k] {
			return xik < x[j].index[k]
		}
	}
	return len(x[i].index) < len(x[j].index)
}

// typeFields returns a list of fields that should be recognized for the given type.
// The algorithm is breadth-first search over the set of structs to include - the top struct
// and then any reachable anonymous structs.
func typeFields(t reflect.Type) []field {
	// Anonymous fields to explore at the current level and the next.
	current := []field{}
	next := []field{{typ: t}}

	// Count of queued names for current level and the next.
	count := map[reflect.Type]int{}
	nextCount := map[reflect.Type]int{}

	// Types already visited at an earlier level.
	visited := map[reflect.Type]bool{}

	// Fields found.
	var fields []field

	for len(next) > 0 {
		current, next = next, current[:0]
		count, nextCount = nextCount, map[reflect.Type]int{}

		for _, f := range current {
			if visited[f.typ] {
				continue
			}
			visited[f.typ] = true

			// Scan f.typ for fields to include.
			for i := 0; i < f.typ.NumField(); i++ {
				sf := f.typ.Field(i)
				isUnexported := sf.PkgPath != ""
				if sf.Anonymous {
					t := sf.Type
					if t.Kind() == reflect.Ptr {
						t = t.Elem()
					}
					if isUnexported && t.Kind() != reflect.Struct {
						// Ignore embedded fields of unexported non-struct types.
						continue
					}
					// Do not ignore embedded fields of unexported struct types
					// since they may have exported fields.
				} else if isUnexported {
					// Ignore unexported non-embedded fields.
					continue
				}
				spec := parseFieldTag(sf)
				if spec == nil && !sf.Anonymous {
					// accept non-anonymous inner struct
					toCont := true
					t := sf.Type
					if t.Kind() == reflect.Ptr {
						t = t.Elem()
						if t.Kind() == reflect.Struct {
							toCont = false
						}
					}
					if toCont {
						continue
					}
				}

				index := make([]int, len(f.index)+1)
				copy(index, f.index)
				index[len(f.index)] = i

				ft := sf.Type
				if ft.Name() == "" && ft.Kind() == reflect.Ptr {
					// Follow pointer.
					ft = ft.Elem()
				}

				// Record found field and index sequence.
				if spec != nil && (spec.getName() != "" || !sf.Anonymous || ft.Kind() != reflect.Struct) {
					if spec.getName() == "" {
						spec.setName(sf.Name)
					}
					fields = append(fields, field{
						name:  spec.getName(),
						index: index,
						typ:   ft,
						spec:  spec,
					})
					if count[f.typ] > 1 {
						// If there were multiple instances, add a second,
						// so that the annihilation code will see a duplicate.
						// It only cares about the distinction between 1 or 2,
						// so don't bother generating any more copies.
						fields = append(fields, fields[len(fields)-1])
					}
					continue
				}

				// Record new anonymous struct to explore in next round.
				nextCount[ft]++
				if nextCount[ft] == 1 {
					next = append(next, field{name: ft.Name(), index: index, typ: ft})
				}
			}
		}
	}

	sort.Slice(fields, func(i, j int) bool {
		x := fields
		// sort field by name, breaking ties with depth, then
		// breaking ties with "name came from cmd tag", then
		// breaking ties with index sequence.
		if x[i].name != x[j].name {
			return x[i].name < x[j].name
		}
		if len(x[i].index) != len(x[j].index) {
			return len(x[i].index) < len(x[j].index)
		}
		return byIndex(x).Less(i, j)
	})

	// Delete all fields that are hidden by the Go rules for embedded fields,
	// except that fields with accepted tags are promoted.

	// The fields are sorted in primary order of name, secondary order
	// of field index length. Loop over names; for each name, delete
	// hidden fields by choosing the one dominant field that survives.
	out := fields[:0]
	for advance, i := 0, 0; i < len(fields); i += advance {
		// One iteration per name.
		// Find the sequence of fields with the name of this first field.
		fi := fields[i]
		name := fi.name
		for advance = 1; i+advance < len(fields); advance++ {
			fj := fields[i+advance]
			if fj.name != name {
				break
			}
		}
		if advance == 1 { // Only one field with this name
			out = append(out, fi)
			continue
		}
		/* ignore multiple fields with same name
		dominant, ok := dominantField(fields[i : i+advance])
		if ok {
			out = append(out, dominant)
		}
		*/
	}

	fields = out
	sort.Sort(byIndex(fields))

	return fields
}

// ----- TAGS
func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allowed
			// in a tag name.
		default:
			if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
				return false
			}
		}
	}
	return true
}

func parseFieldTag(sf reflect.StructField) cmdSpec {
	tag := sf.Tag.Get(cmdTag)
	if tag == "-" || tag == "" {
		return nil
	}
	kind, opts := parseTag(tag)
	if !isValidTag(kind) {
		kind = ""
	}
	name := opts.At(0)
	description := opts.At(1)

	if name == "" {
		name = sf.Name
	}

	var annotations []string
	annot := strings.Trim(sf.Tag.Get(metaTag), " ")
	if annot != "" {
		annotations = splitString(annot)
	}

	switch kind {
	case "":
		fallthrough //default to 'arg'
	case argTag:
		order, err := strconv.Atoi(opts.At(2))
		if err != nil {
			order = -1
		}
		return &argSpec{
			name:        name,
			description: description,
			order:       order,
			annotations: annotations,
		}
	case flagTag:
		persistent, _ := strconv.ParseBool(opts.At(3))
		required, _ := strconv.ParseBool(opts.At(4))
		hidden, _ := strconv.ParseBool(opts.At(5))
		return &flagSpec{
			name:        name,
			description: description,
			shorthand:   opts.At(2),
			persistent:  persistent,
			required:    required,
			hidden:      hidden,
			annotations: annotations,
		}
	default:
		return nil
	}
}

func splitString(s string) []string {
	ret := strings.Split(s, ",")
	for i, r := range ret {
		ret[i] = strings.Trim(r, " ")
	}
	return ret
}

// tagOptions is the string following a comma in a struct field's "cmd" tag
// or the empty array.
type tagOptions []string

// parseTag splits a struct field's cmd tag into its type and comma-separated
// options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], splitString(tag[idx+1:])
	}
	return tag, []string{}
}

// Contains reports whether a given option was set
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	for _, s := range []string(o) {
		if s == optionName {
			return true
		}
	}
	return false
}

func (o tagOptions) At(i int) string {
	if i < 0 || i >= len(o) {
		return ""
	}
	return o[i]
}
